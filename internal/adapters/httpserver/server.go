package httpserver

import (
	"net/http"

	"bug-report-service/internal/adapters/config"
	"bug-report-service/internal/adapters/httpserver/middleware"
	"bug-report-service/internal/adapters/observability"

	"github.com/go-chi/chi/v5"
)

func NewServerWithHandler(cfg config.Config, log observability.Logger, h http.Handler) *http.Server {

	r := chi.NewRouter()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logging(log))
	r.Use(middleware.CORS(cfg.CORS.AllowedOrigins))
	r.Use(middleware.RateLimit(cfg.RateLimit.RPS, cfg.RateLimit.Burst))

	r.Mount("/", h)

	return &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      r,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}
}
