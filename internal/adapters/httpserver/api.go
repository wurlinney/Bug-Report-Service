package httpserver

import (
	"encoding/json"
	"net/http"

	"bug-report-service/internal/application/attachment"
	"bug-report-service/internal/application/auth"
	"bug-report-service/internal/application/ports"
	"bug-report-service/internal/application/report"
	"bug-report-service/internal/application/user"

	"github.com/go-chi/chi/v5"
)

type Principal struct {
	UserID string
	Role   string
}

type TokenVerifier interface {
	VerifyAccessToken(token string) (Principal, error)
}

type Deps struct {
	Ready Readiness

	AuthService       *auth.Service
	UserService       *user.Service
	ReportService     *report.Service
	AttachmentService *attachment.Service
	AttachmentSigner  ports.ObjectURLSigner
	TokenVerifier     TokenVerifier

	TusUploads http.Handler
}

func NewAPI(deps Deps) http.Handler {
	r := chi.NewRouter()

	// health/readiness
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		status, payload := deps.Ready.ReadyResponse()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(payload)
	})

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", registerHandler(deps))
			r.Post("/login", loginHandler(deps))
			r.Post("/refresh", refreshHandler(deps))
		})

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(deps.TokenVerifier))
			r.Get("/me", meHandler(deps))
			r.Post("/reports", createReportHandler(deps))
			r.Get("/reports", listMyReportsHandler(deps))
			r.Get("/reports/{id}", getMyReportHandler(deps))
			r.Get("/reports/{id}/attachments", listReportAttachmentsHandler(deps))
			if deps.TusUploads != nil {
				r.With(TusCreateGuard(deps)).Mount("/uploads/", deps.TusUploads)
			}
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
