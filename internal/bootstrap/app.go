package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"bug-report-service/internal/adapters/config"
	"bug-report-service/internal/adapters/httpserver"
	"bug-report-service/internal/adapters/observability"
	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/adapters/security"
	s3adapter "bug-report-service/internal/adapters/storage/s3"
	"bug-report-service/internal/application/attachment"
	"bug-report-service/internal/application/auth"
	"bug-report-service/internal/application/report"
	"bug-report-service/internal/application/user"

	"github.com/jackc/pgx/v5/pgxpool"
	tushandler "github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/s3store"
)

type App struct {
	cfg    config.Config
	logger observability.Logger

	httpServer *http.Server
	ready      httpserver.Readiness

	db *pgxpool.Pool
}

func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger, err := observability.NewLogger(cfg)
	if err != nil {
		return nil, err
	}

	ready := httpserver.NewReadiness()

	apiDeps := httpserver.Deps{Ready: ready}

	var db *pgxpool.Pool
	// Full mode only when both DB and JWT secret are configured.
	if cfg.DB.URL != "" && cfg.JWT.Secret != "" {
		pool, err := pgxpool.New(context.Background(), cfg.DB.URL)
		if err != nil {
			ready.SetDependency("db", false)
			return nil, err
		}
		db = pool
		ready.SetDependency("db", true)

		usersRepo := postgres.NewUserRepository(db)
		rtRepo := postgres.NewRefreshTokenRepository(db)
		reportsRepo := postgres.NewReportRepository(db)
		attsRepo := postgres.NewAttachmentRepository(db)

		hasher := security.NewBCryptPasswordHasher(12)
		jwtMgr := security.NewJWTManager(security.JWTConfig{
			Issuer:        cfg.JWT.Issuer,
			AccessTTL:     cfg.JWT.AccessTTL,
			RefreshTTL:    cfg.JWT.RefreshTTL,
			HMACSecretKey: []byte(cfg.JWT.Secret),
		})

		authSvc := auth.NewService(auth.Deps{
			Users:         usersRepo,
			RefreshTokens: rtRepo,
			Hasher:        hasher,
			JWT:           jwtMgr,
			Random:        security.NewTokenGenerator(),
			Clock:         security.RealClock{},
			RefreshTTL:    cfg.JWT.RefreshTTL,
		})

		userSvc := user.NewService(usersRepo)
		reportSvc := report.NewService(report.Deps{
			Reports: reportsRepo,
			Clock:   security.RealClock{},
			Random:  security.NewTokenGenerator(),
		})
		const uploadMaxSize = 20 * 1024 * 1024
		allowedMIMEs := map[string]struct{}{"image/png": {}, "image/jpeg": {}, "image/webp": {}}

		attSvc := attachment.NewService(attachment.Deps{
			Reports:      reportsRepo,
			Attachments:  attsRepo,
			Storage:      nil,
			Clock:        security.RealClock{},
			Random:       security.NewTokenGenerator(),
			MaxFileSize:  uploadMaxSize,
			AllowedMIMEs: allowedMIMEs,
		})

		apiDeps.AuthService = authSvc
		apiDeps.UserService = userSvc
		apiDeps.ReportService = reportSvc
		apiDeps.AttachmentService = attSvc
		apiDeps.TokenVerifier = security.TokenVerifier{JWT: jwtMgr}

		s3c, err := s3adapter.NewClient(context.Background(), cfg)
		if err != nil {
			ready.SetDependency("s3", false)
			return nil, err
		}
		ready.SetDependency("s3", true)

		store := s3store.New(cfg.S3.Bucket, s3c)
		store.ObjectPrefix = "tus"
		store.MetadataObjectPrefix = "tus-meta"
		composer := tushandler.NewStoreComposer()
		store.UseIn(composer)
		tus, err := tushandler.NewHandler(tushandler.Config{
			BasePath:                "/api/v1/uploads/",
			MaxSize:                 uploadMaxSize,
			StoreComposer:           composer,
			NotifyCompleteUploads:   true,
			PreUploadCreateCallback: httpserver.TusPreUploadCreateCallback(apiDeps, uploadMaxSize, allowedMIMEs),
		})
		if err == nil {
			apiDeps.TusUploads = tus
			go func() {
				for ev := range tus.CompleteUploads {
					meta := ev.Upload.MetaData
					reportID := strings.TrimSpace(meta["report_id"])
					if reportID == "" {
						continue
					}
					uploaderID := strings.TrimSpace(meta["uploader_id"])
					uploaderRole := strings.TrimSpace(meta["uploader_role"])
					if uploaderID == "" || uploaderRole == "" {
						continue
					}
					_, _ = attSvc.Finalize(context.Background(), attachment.FinalizeRequest{
						ActorRole:      uploaderRole,
						ActorID:        uploaderID,
						ReportID:       reportID,
						UploadID:       ev.Upload.ID,
						FileName:       meta["filename"],
						ContentType:    meta["content_type"],
						FileSize:       ev.Upload.Size,
						StorageKey:     "tus/" + ev.Upload.ID,
						IdempotencyKey: meta["idempotency_key"],
					})
				}
			}()
		}
	}

	api := httpserver.NewAPI(apiDeps)
	srv := httpserver.NewServerWithHandler(cfg, logger, api)

	return &App{
		cfg:        cfg,
		logger:     logger,
		httpServer: srv,
		ready:      ready,
		db:         db,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting api", "addr", a.cfg.HTTP.Addr, "env", a.cfg.AppEnv)

	errCh := make(chan error, 1)
	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			a.logger.Error("http server stopped with error", "err", err.Error())
			return err
		}
		a.logger.Info("http server stopped")
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a.ready.SetShuttingDown()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("http server shutdown error", "err", err.Error())
		return err
	}

	if a.db != nil {
		a.db.Close()
	}

	a.logger.Info("shutdown complete")
	return nil
}
