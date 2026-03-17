package httpserver

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"bug-report-service/internal/application/attachment"
	"bug-report-service/internal/application/auth"
	"bug-report-service/internal/application/message"
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
	MessageService    *message.Service
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
			r.Get("/reports/{id}/messages", listReportMessagesHandler(deps))
			r.Post("/reports/{id}/messages", createReportMessageHandler(deps))
			r.Get("/reports/{id}/attachments", listReportAttachmentsHandler(deps))
			if deps.TusUploads != nil {
				const tusBase = "/api/v1/uploads"
				// tusd routes based on r.URL.Path. Strip API prefix so tusd sees "/" (create) or "/<id>".
				tusInner := http.StripPrefix(tusBase, deps.TusUploads)
				tus := withTusLocationRewrite(tusBase, tusInner)
				r.With(TusCreateGuard(deps)).Handle("/uploads", tus)
				r.With(TusCreateGuard(deps)).Handle("/uploads/*", tus)
			}
		})

		r.Route("/mod", func(r chi.Router) {
			r.Use(AuthMiddleware(deps.TokenVerifier))
			r.Use(ModeratorOnly)
			r.Get("/reports", listAllReportsHandler(deps))
			r.Get("/reports/{id}", getReportHandler(deps))
			r.Patch("/reports/{id}/status", changeReportStatusHandler(deps))
			r.Get("/reports/{id}/messages", listReportMessagesHandler(deps))
			r.Post("/reports/{id}/messages", createReportMessageHandler(deps))
			r.Get("/reports/{id}/attachments", listReportAttachmentsHandler(deps))
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type locationRewriteRW struct {
	http.ResponseWriter
	tusBase string
}

func (w *locationRewriteRW) WriteHeader(statusCode int) {
	// If tusd returns Location like "/<id>", rewrite to "/api/v1/uploads/<id>".
	loc := w.Header().Get("Location")
	if statusCode == http.StatusCreated && loc != "" {
		// Relative location
		if strings.HasPrefix(loc, "/") && !strings.HasPrefix(loc, w.tusBase+"/") {
			w.Header().Set("Location", w.tusBase+loc)
		} else if strings.HasPrefix(loc, "http://") || strings.HasPrefix(loc, "https://") {
			// Absolute location
			if u, err := url.Parse(loc); err == nil && strings.HasPrefix(u.Path, "/") && !strings.HasPrefix(u.Path, w.tusBase+"/") {
				u.Path = w.tusBase + u.Path
				w.Header().Set("Location", u.String())
			}
		}
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func withTusLocationRewrite(tusBase string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&locationRewriteRW{ResponseWriter: w, tusBase: tusBase}, r)
	})
}
