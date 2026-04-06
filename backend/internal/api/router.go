package api

import (
	"net/http"

	changereportmeta "bug-report-service/internal/api/handler/change_report_meta"
	changereportstatus "bug-report-service/internal/api/handler/change_report_status"
	createnote "bug-report-service/internal/api/handler/create_note"
	createreport "bug-report-service/internal/api/handler/create_report"
	createuploadsession "bug-report-service/internal/api/handler/create_upload_session"
	deleteupload "bug-report-service/internal/api/handler/delete_upload"
	getreport "bug-report-service/internal/api/handler/get_report"
	listattachments "bug-report-service/internal/api/handler/list_attachments"
	listnotes "bug-report-service/internal/api/handler/list_notes"
	listreports "bug-report-service/internal/api/handler/list_reports"
	modlogin "bug-report-service/internal/api/handler/mod_login"
	modprofile "bug-report-service/internal/api/handler/mod_profile"
	modrefresh "bug-report-service/internal/api/handler/mod_refresh"
	"bug-report-service/internal/api/handler/tus"
	"bug-report-service/internal/infrastructure/middleware"

	"github.com/go-chi/chi/v5"
)

type Deps struct {
	Ready         Readiness
	TokenVerifier TokenVerifier

	CreateUploadSession createuploadsession.UseCase
	DeleteUpload        deleteupload.UseCase
	DeleteUploadSession deleteupload.SessionChecker
	CreateReport        createreport.UseCase
	ModLogin            modlogin.UseCase
	ModRefresh          modrefresh.UseCase
	ModProfile          modprofile.UseCase
	ListReports         listreports.UseCase
	GetReport           getreport.UseCase
	ChangeReportStatus  changereportstatus.StatusUseCase
	ChangeReportMeta    changereportmeta.UseCase
	CreateNote          createnote.UseCase
	ListNotes           listnotes.UseCase
	ListAttachments     listattachments.UseCase

	TusSessionChecker tus.SessionChecker
	TusUploads        http.Handler

	PublicCreateRPS   float64
	PublicCreateBurst int
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	// health/readiness
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	r.Get("/livez", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		status, payload := deps.Ready.ReadyResponse()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(payload)
	})

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/public", func(r chi.Router) {
			r.Post("/upload-sessions", createuploadsession.New(deps.CreateUploadSession))
			r.Delete("/upload-sessions/{id}/uploads/{uploadId}", deleteupload.New(deps.DeleteUpload, deps.DeleteUploadSession))
			r.With(middleware.RateLimit(deps.PublicCreateRPS, deps.PublicCreateBurst)).Post("/reports", createreport.New(deps.CreateReport))
		})

		r.Route("/mod/auth", func(r chi.Router) {
			r.Post("/login", modlogin.New(deps.ModLogin))
			r.Post("/refresh", modrefresh.New(deps.ModRefresh))
		})

		if deps.TusUploads != nil {
			const tusBase = "/api/v1/uploads"
			tusInner := http.StripPrefix(tusBase, deps.TusUploads)
			tusWrapped := tus.WithTusLocationRewrite(tusBase, tusInner)
			r.With(tus.TusCreateGuard(deps.TusSessionChecker)).Handle("/uploads", tusWrapped)
			r.With(tus.TusCreateGuard(deps.TusSessionChecker)).Handle("/uploads/*", tusWrapped)
		}

		r.Route("/mod", func(r chi.Router) {
			r.Use(AuthMiddleware(deps.TokenVerifier))
			r.Use(ModeratorOnly)
			r.Get("/me", modprofile.New(deps.ModProfile))
			r.Get("/reports", listreports.New(deps.ListReports))
			r.Get("/reports/{id}", getreport.New(deps.GetReport))
			r.Patch("/reports/{id}/status", changereportstatus.New(deps.ChangeReportStatus, deps.ChangeReportMeta))
			r.Patch("/reports/{id}/meta", changereportmeta.New(deps.ChangeReportMeta))
			r.Get("/reports/{id}/notes", listnotes.New(deps.ListNotes))
			r.Post("/reports/{id}/notes", createnote.New(deps.CreateNote))
			r.Get("/reports/{id}/attachments", listattachments.New(deps.ListAttachments))
		})
	})

	return r
}
