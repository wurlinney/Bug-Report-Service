package httpserver

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// DELETE /api/v1/public/upload-sessions/{id}/uploads/{uploadId}
// Removes a previously uploaded file from the upload session so it won't be bound to a report.
func deleteUploadFromSessionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.UploadSessionService == nil || deps.AttachmentService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		uploadID := strings.TrimSpace(chi.URLParam(r, "uploadId"))
		if sessionID == "" || uploadID == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "id and uploadId are required")
			return
		}

		ok, err := deps.UploadSessionService.Exists(r.Context(), sessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}

		storageKey := "tus/" + uploadID
		deleted, err := deps.AttachmentService.DeleteFromSessionByStorageKey(r.Context(), sessionID, storageKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}
		if !deleted {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}
