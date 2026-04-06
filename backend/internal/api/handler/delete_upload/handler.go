package delete_upload

import (
	"net/http"

	"bug-report-service/internal/api/shared"

	"github.com/go-chi/chi/v5"
)

func New(useCase UseCase, sessions SessionChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "id")
		uploadID := chi.URLParam(r, "uploadId")

		if sessionID == "" || uploadID == "" {
			shared.WriteError(w, http.StatusBadRequest, "validation_error", "missing path parameters")
			return
		}

		exists, err := sessions.Exists(r.Context(), sessionID)
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}
		if !exists {
			shared.WriteError(w, http.StatusNotFound, "not_found", "upload session not found")
			return
		}

		storageKey := "tus/" + uploadID

		deleted, err := useCase.Execute(r.Context(), sessionID, storageKey)
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}
		if !deleted {
			shared.WriteError(w, http.StatusNotFound, "not_found", "upload not found")
			return
		}

		shared.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
