package httpserver

import (
	"net/http"
	"strings"
	"time"

	"bug-report-service/internal/application/attachment"

	"github.com/go-chi/chi/v5"
)

func listReportAttachmentsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if deps.AttachmentService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}

		reportID := strings.TrimSpace(chi.URLParam(r, "id"))
		items, err := deps.AttachmentService.ListForReport(r.Context(), attachment.ListForReportRequest{
			ActorRole: p.Role,
			ActorID:   p.UserID,
			ReportID:  reportID,
		})
		if err != nil {
			switch err {
			case attachment.ErrBadInput:
				writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
			case attachment.ErrNotFound:
				writeError(w, http.StatusNotFound, "not_found", "not found")
			case attachment.ErrForbidden:
				writeError(w, http.StatusForbidden, "forbidden", "forbidden")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			}
			return
		}

		out := make([]map[string]any, 0, len(items))
		for _, it := range items {
			obj := map[string]any{
				"id":           it.ID,
				"report_id":    it.ReportID,
				"file_name":    it.FileName,
				"content_type": it.ContentType,
				"file_size":    it.FileSize,
				"storage_key":  it.StorageKey,
				"created_at":   it.CreatedAt.Unix(),
				"download_url": "",
			}
			if deps.AttachmentSigner != nil {
				if url, err := deps.AttachmentSigner.PresignGetObject(r.Context(), it.StorageKey, 15*time.Minute); err == nil {
					obj["download_url"] = url
				}
			}
			out = append(out, obj)
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}
