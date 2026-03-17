package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"bug-report-service/internal/adapters/config"
	"bug-report-service/internal/adapters/httpserver"
	"bug-report-service/internal/adapters/observability"
)

type App struct {
	cfg    config.Config
	logger observability.Logger

	httpServer *http.Server
	ready      httpserver.Readiness
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

	srv := httpserver.NewServer(cfg, logger, ready)

	return &App{
		cfg:        cfg,
		logger:     logger,
		httpServer: srv,
		ready:      ready,
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

	a.logger.Info("shutdown complete")
	return nil
}
