package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"bug-report-service/internal/api"
	"bug-report-service/internal/infrastructure/auth"
	"bug-report-service/internal/infrastructure/config"
	dbattachment "bug-report-service/internal/infrastructure/database/attachment"
	dbnote "bug-report-service/internal/infrastructure/database/note"
	dbrefreshtoken "bug-report-service/internal/infrastructure/database/refreshtoken"
	dbreport "bug-report-service/internal/infrastructure/database/report"
	dbuploadsession "bug-report-service/internal/infrastructure/database/uploadsession"
	dbuser "bug-report-service/internal/infrastructure/database/user"
	"bug-report-service/internal/infrastructure/logger"
	s3adapter "bug-report-service/internal/infrastructure/storage/s3"
	changereportmeta "bug-report-service/internal/usecase/change_report_meta"
	changereportstatus "bug-report-service/internal/usecase/change_report_status"
	createnote "bug-report-service/internal/usecase/create_note"
	createreport "bug-report-service/internal/usecase/create_report"
	createuploadsession "bug-report-service/internal/usecase/create_upload_session"
	deleteupload "bug-report-service/internal/usecase/delete_upload"
	finalizeattachment "bug-report-service/internal/usecase/finalize_attachment"
	getreport "bug-report-service/internal/usecase/get_report"
	listattachments "bug-report-service/internal/usecase/list_attachments"
	listnotes "bug-report-service/internal/usecase/list_notes"
	listreports "bug-report-service/internal/usecase/list_reports"
	modlogin "bug-report-service/internal/usecase/mod_login"
	modprofile "bug-report-service/internal/usecase/mod_profile"
	modrefresh "bug-report-service/internal/usecase/mod_refresh"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	tushandler "github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/s3store"
)

type App struct {
	cfg    config.Config
	logger logger.Logger

	httpServer *http.Server
	ready      api.Readiness

	db *pgxpool.Pool
}

func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	log, err := logger.NewLogger(cfg)
	if err != nil {
		return nil, err
	}

	ready := api.NewReadiness()

	apiDeps := api.Deps{Ready: ready}

	var db *pgxpool.Pool
	if cfg.DB.URL != "" && cfg.JWT.Secret != "" {
		pool, err := pgxpool.New(context.Background(), cfg.DB.URL)
		if err != nil {
			ready.SetDependency("db", false)
			return nil, err
		}
		db = pool
		ready.SetDependency("db", true)

		usersRepo := dbuser.NewRepository(db)
		rtRepo := dbrefreshtoken.NewRepository(db)
		reportsRepo := dbreport.NewRepository(db)
		uploadSessionsRepo := dbuploadsession.NewRepository(db)
		attsRepo := dbattachment.NewRepository(db)
		notesRepo := dbnote.NewRepository(db)

		jwtMgr := auth.NewJWTManager(auth.JWTConfig{
			Issuer:        cfg.JWT.Issuer,
			AccessTTL:     cfg.JWT.AccessTTL,
			RefreshTTL:    cfg.JWT.RefreshTTL,
			HMACSecretKey: []byte(cfg.JWT.Secret),
		})
		hasher := auth.NewBCryptPasswordHasher(12)
		rng := auth.NewTokenGenerator()
		clock := auth.RealClock{}

		seedModerators(context.Background(), log, usersRepo, hasher, cfg.ModeratorSeed)

		sessionChecker := &sessionCheckerAdapter{repo: uploadSessionsRepo}

		// Use cases
		createReportUC := createreport.New(reportsRepo, sessionChecker, attsRepo)
		listReportsUC := listreports.New(reportsRepo)
		getReportUC := getreport.New(reportsRepo)
		changeStatusUC := changereportstatus.New(reportsRepo)
		changeMetaUC := changereportmeta.New(reportsRepo)
		loginUC := modlogin.New(usersRepo, hasher, jwtMgr, rtRepo, rng, clock, cfg.JWT.RefreshTTL)
		refreshUC := modrefresh.New(rtRepo, jwtMgr, rng, clock, cfg.JWT.RefreshTTL)
		profileUC := modprofile.New(usersRepo)
		createNoteUC := createnote.New(notesRepo, reportsRepo)
		listNotesUC := listnotes.New(notesRepo, reportsRepo)
		createUploadSessionUC := createuploadsession.New(uploadSessionsRepo)
		deleteUploadUC := deleteupload.New(attsRepo)

		apiDeps.CreateReport = createReportUC
		apiDeps.ListReports = listReportsUC
		apiDeps.GetReport = getReportUC
		apiDeps.ChangeReportStatus = changeStatusUC
		apiDeps.ChangeReportMeta = changeMetaUC
		apiDeps.ModLogin = loginUC
		apiDeps.ModRefresh = refreshUC
		apiDeps.ModProfile = profileUC
		apiDeps.CreateNote = createNoteUC
		apiDeps.ListNotes = listNotesUC
		apiDeps.CreateUploadSession = createUploadSessionUC
		apiDeps.DeleteUpload = deleteUploadUC
		apiDeps.DeleteUploadSession = sessionChecker
		apiDeps.TokenVerifier = auth.TokenVerifier{JWT: jwtMgr}
		apiDeps.PublicCreateRPS = cfg.RateLimit.RPS
		apiDeps.PublicCreateBurst = cfg.RateLimit.Burst

		s3c, err := s3adapter.NewClient(context.Background(), cfg)
		if err != nil {
			ready.SetDependency("s3", false)
			return nil, err
		}
		ready.SetDependency("s3", true)

		presignClient := s3c
		if strings.TrimSpace(cfg.S3.PublicEndpoint) != "" && strings.TrimSpace(cfg.S3.PublicEndpoint) != strings.TrimSpace(cfg.S3.Endpoint) {
			publicS3c, err := s3adapter.NewPublicClient(context.Background(), cfg)
			if err != nil {
				ready.SetDependency("s3", false)
				return nil, err
			}
			presignClient = publicS3c
		}

		signer := s3adapter.NewPresigner(cfg.S3.Bucket, presignClient)
		apiDeps.ListAttachments = listattachments.New(reportsRepo, attsRepo, signer)

		const uploadMaxSize = 20 * 1024 * 1024
		allowedMIMEs := map[string]struct{}{
			"image/png":         {},
			"image/jpeg":        {},
			"image/jpg":         {},
			"image/webp":        {},
			"application/pdf":   {},
			"application/x-pdf": {},
		}

		finalizeUC := finalizeattachment.New(uploadSessionsRepo, attsRepo, attsRepo, uploadMaxSize, allowedMIMEs)

		apiDeps.TusSessionChecker = sessionChecker

		if cfg.TusCleanup.Enabled {
			startTusOrphanCleanup(log, s3c, cfg.S3.Bucket, attsRepo, cfg.TusCleanup.ObjectPrefix, cfg.TusCleanup.GracePeriod, cfg.TusCleanup.Interval)
		}

		store := s3store.New(cfg.S3.Bucket, s3c)
		store.ObjectPrefix = "tus"
		store.MetadataObjectPrefix = "tus-meta"
		composer := tushandler.NewStoreComposer()
		store.UseIn(composer)
		tusH, err := tushandler.NewHandler(tushandler.Config{
			BasePath:                "/",
			MaxSize:                 uploadMaxSize,
			StoreComposer:           composer,
			NotifyCompleteUploads:   true,
			PreUploadCreateCallback: tusdPreUploadCallback(sessionChecker, uploadMaxSize, allowedMIMEs),
		})
		if err == nil {
			apiDeps.TusUploads = tusH
			go func() {
				for ev := range tusH.CompleteUploads {
					meta := ev.Upload.MetaData
					uploadSessionID := strings.TrimSpace(meta["upload_session_id"])
					if uploadSessionID == "" {
						log.Error("tus finalize: missing upload_session_id in metadata", "upload_id", ev.Upload.ID)
						continue
					}
					storageKey := strings.TrimSpace(ev.Upload.Storage["Key"])
					if storageKey == "" {
						storageKey = "tus/" + ev.Upload.ID
					}
					_, fErr := finalizeUC.Execute(context.Background(), finalizeattachment.Request{
						UploadSessionID: uploadSessionID,
						UploadID:        ev.Upload.ID,
						FileName:        meta["filename"],
						ContentType:     meta["content_type"],
						FileSize:        ev.Upload.Size,
						StorageKey:      storageKey,
						IdempotencyKey:  meta["idempotency_key"],
					})
					if fErr != nil {
						log.Error("tus finalize: failed to persist attachment",
							"upload_id", ev.Upload.ID,
							"upload_session_id", uploadSessionID,
							"error", fErr.Error(),
						)
						key := "tus/" + ev.Upload.ID
						_, delErr := s3c.DeleteObject(context.Background(), &s3.DeleteObjectInput{
							Bucket: &cfg.S3.Bucket,
							Key:    &key,
						})
						if delErr != nil {
							log.Error("tus finalize: failed to cleanup object in s3",
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

	router := api.NewRouter(apiDeps)
	srv := api.NewServerWithHandler(cfg, log, router)

	return &App{
		cfg:        cfg,
		logger:     log,
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

type sessionCheckerAdapter struct {
	repo *dbuploadsession.Repository
}

func (a *sessionCheckerAdapter) Exists(ctx context.Context, id string) (bool, error) {
	_, found, err := a.repo.GetByID(ctx, id)
	return found, err
}

// sessionGetterForCreateReport adapts sessionCheckerAdapter to create_report.SessionGetter.
func (a *sessionCheckerAdapter) GetByID(ctx context.Context, id string) (bool, error) {
	_, found, err := a.repo.GetByID(ctx, id)
	return found, err
}

func startTusOrphanCleanup(
	log logger.Logger,
	s3c *s3.Client,
	bucket string,
	attachments interface {
		ExistsByStorageKey(ctx context.Context, storageKey string) (bool, error)
	},
	prefix string,
	gracePeriod time.Duration,
	interval time.Duration,
) {
	run := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cleanupTusOrphans(ctx, log, s3c, bucket, attachments, prefix, gracePeriod)
	}

	go func() {
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
	log logger.Logger,
	s3c *s3.Client,
	bucket string,
	attachments interface {
		ExistsByStorageKey(ctx context.Context, storageKey string) (bool, error)
	},
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
			log.Error("tus cleanup: list objects failed", "error", err.Error())
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
				log.Error("tus cleanup: db check failed", "key", key, "error", err.Error())
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
				log.Error("tus cleanup: delete object failed", "key", key, "error", err.Error())
				continue
			}
			deleted++
		}
	}

	if deleted > 0 {
		log.Info("tus cleanup: orphan objects deleted", "count", deleted)
	}
}

func tusdPreUploadCallback(
	sessions interface {
		Exists(ctx context.Context, id string) (bool, error)
	},
	maxSize int64,
	allowedMIMEs map[string]struct{},
) func(hook tushandler.HookEvent) (tushandler.HTTPResponse, tushandler.FileInfoChanges, error) {
	return func(hook tushandler.HookEvent) (tushandler.HTTPResponse, tushandler.FileInfoChanges, error) {
		if maxSize > 0 && hook.Upload.Size > maxSize {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("file_too_large", "file too large", http.StatusRequestEntityTooLarge)
		}

		meta := hook.Upload.MetaData
		uploadSessionID := strings.TrimSpace(meta["upload_session_id"])
		filename := strings.TrimSpace(meta["filename"])
		contentType := strings.TrimSpace(meta["content_type"])
		if uploadSessionID == "" || filename == "" || contentType == "" {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("validation_error", "upload_session_id, filename and content_type are required", http.StatusBadRequest)
		}
		if _, ok := allowedMIMEs[contentType]; !ok {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("unsupported_media_type", "unsupported content type", http.StatusBadRequest)
		}

		ok, err := sessions.Exists(hook.Context, uploadSessionID)
		if err != nil {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("internal_error", "internal error", http.StatusInternalServerError)
		}
		if !ok {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("not_found", "not found", http.StatusNotFound)
		}

		newMeta := make(tushandler.MetaData, len(meta)+1)
		for k, v := range meta {
			newMeta[k] = v
		}
		newMeta["filetype"] = contentType

		return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{MetaData: newMeta}, nil
	}
}
