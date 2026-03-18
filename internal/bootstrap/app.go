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
	"bug-report-service/internal/application/moderator"
	"bug-report-service/internal/application/note"
	"bug-report-service/internal/application/ports"
	"bug-report-service/internal/application/report"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

		usersRepo := postgres.NewModeratorRepository(db)
		rtRepo := postgres.NewRefreshTokenRepository(db)
		reportsRepo := postgres.NewReportRepository(db)
		attsRepo := postgres.NewAttachmentRepository(db)
		notesRepo := postgres.NewNoteRepository(db)
		jwtMgr := security.NewJWTManager(security.JWTConfig{
			Issuer:        cfg.JWT.Issuer,
			AccessTTL:     cfg.JWT.AccessTTL,
			RefreshTTL:    cfg.JWT.RefreshTTL,
			HMACSecretKey: []byte(cfg.JWT.Secret),
		})
		hasher := security.NewBCryptPasswordHasher(12)
		authSvc := auth.NewService(auth.Deps{
			Users:         usersRepo,
			RefreshTokens: rtRepo,
			Hasher:        hasher,
			JWT:           jwtMgr,
			Random:        security.NewTokenGenerator(),
			Clock:         security.RealClock{},
			RefreshTTL:    cfg.JWT.RefreshTTL,
		})
		modSvc := moderator.NewService(usersRepo)
		noteSvc := note.NewService(note.Deps{
			Notes:   notesRepo,
			Reports: reportsRepo,
			Clock:   security.RealClock{},
			Random:  security.NewTokenGenerator(),
		})

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

		apiDeps.ModAuthService = authSvc
		apiDeps.ModeratorService = modSvc
		apiDeps.NoteService = noteSvc
		apiDeps.ReportService = reportSvc
		apiDeps.AttachmentService = attSvc
		apiDeps.TokenVerifier = security.TokenVerifier{JWT: jwtMgr}
		apiDeps.PublicCreateRPS = cfg.RateLimit.RPS
		apiDeps.PublicCreateBurst = cfg.RateLimit.Burst

		s3c, err := s3adapter.NewClient(context.Background(), cfg)
		if err != nil {
			ready.SetDependency("s3", false)
			return nil, err
		}
		ready.SetDependency("s3", true)

		apiDeps.AttachmentSigner = s3adapter.NewPresigner(cfg.S3.Bucket, s3c)
		if cfg.TusCleanup.Enabled {
			startTusOrphanCleanup(
				logger,
				s3c,
				cfg.S3.Bucket,
				attsRepo,
				cfg.TusCleanup.ObjectPrefix,
				cfg.TusCleanup.GracePeriod,
				cfg.TusCleanup.Interval,
			)
		}

		store := s3store.New(cfg.S3.Bucket, s3c)
		store.ObjectPrefix = "tus"
		store.MetadataObjectPrefix = "tus-meta"
		composer := tushandler.NewStoreComposer()
		store.UseIn(composer)
		tus, err := tushandler.NewHandler(tushandler.Config{
			// We route /api/v1/uploads via chi and rewrite Location header there.
			BasePath:                "/",
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
						logger.Error("tus finalize: missing report_id in metadata", "upload_id", ev.Upload.ID)
						continue
					}
					_, fErr := attSvc.Finalize(context.Background(), attachment.FinalizeRequest{
						ReportID:       reportID,
						UploadID:       ev.Upload.ID,
						FileName:       meta["filename"],
						ContentType:    meta["content_type"],
						FileSize:       ev.Upload.Size,
						StorageKey:     "tus/" + ev.Upload.ID,
						IdempotencyKey: meta["idempotency_key"],
					})
					if fErr != nil {
						logger.Error("tus finalize: failed to persist attachment",
							"upload_id", ev.Upload.ID,
							"report_id", reportID,
							"error", fErr.Error(),
						)
						// Best-effort cleanup to avoid leaving untracked objects in S3.
						key := "tus/" + ev.Upload.ID
						_, delErr := s3c.DeleteObject(context.Background(), &s3.DeleteObjectInput{
							Bucket: &cfg.S3.Bucket,
							Key:    &key,
						})
						if delErr != nil {
							logger.Error("tus finalize: failed to cleanup object in s3",
								"upload_id", ev.Upload.ID,
								"key", key,
								"error", delErr.Error(),
							)
						}
					}
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

func startTusOrphanCleanup(
	logger observability.Logger,
	s3c *s3.Client,
	bucket string,
	attachments ports.AttachmentRepository,
	prefix string,
	gracePeriod time.Duration,
	interval time.Duration,
) {
	run := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cleanupTusOrphans(ctx, logger, s3c, bucket, attachments, prefix, gracePeriod)
	}

	go func() {
		// Startup pass to clean up leftovers after crashes/restarts.
		run()
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			run()
		}
	}()
}

func cleanupTusOrphans(
	ctx context.Context,
	logger observability.Logger,
	s3c *s3.Client,
	bucket string,
	attachments ports.AttachmentRepository,
	prefix string,
	gracePeriod time.Duration,
) {
	cutoff := time.Now().Add(-gracePeriod)
	pager := s3.NewListObjectsV2Paginator(s3c, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	var deleted int
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error("tus cleanup: list objects failed", "error", err.Error())
			return
		}

		for _, obj := range page.Contents {
			if obj.Key == nil || obj.LastModified == nil {
				continue
			}
			if obj.LastModified.After(cutoff) {
				continue
			}

			key := *obj.Key
			found, err := attachments.ExistsByStorageKey(ctx, key)
			if err != nil {
				logger.Error("tus cleanup: db check failed", "key", key, "error", err.Error())
				continue
			}
			if found {
				continue
			}

			_, err = s3c.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				logger.Error("tus cleanup: delete object failed", "key", key, "error", err.Error())
				continue
			}
			deleted++
		}
	}

	if deleted > 0 {
		logger.Info("tus cleanup: orphan objects deleted", "count", deleted)
	}
}
