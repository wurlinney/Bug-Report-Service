package list_attachments

import (
	"net/http"

	"bug-report-service/internal/api/shared"
	uc "bug-report-service/internal/usecase/list_attachments"

	"github.com/go-chi/chi/v5"
)

type attachmentItem struct {
	ID              int64  `json:"id"`
	ReportID        string `json:"report_id"`
	UploadSessionID string `json:"upload_session_id"`
	FileName        string `json:"file_name"`
	ContentType     string `json:"content_type"`
	FileSize        int64  `json:"file_size"`
	StorageKey      string `json:"storage_key"`
	DownloadURL     string `json:"download_url"`
	CreatedAt       int64  `json:"created_at"`
}

type responseBody struct {
	Items []attachmentItem `json:"items"`
}

func New(useCase UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := shared.RequirePrincipal(w, r)
		if !ok {
			return
		}

		reportID := chi.URLParam(r, "id")
		if reportID == "" {
			shared.WriteError(w, http.StatusBadRequest, "validation_error", "missing report id")
			return
		}

		items, err := useCase.Execute(r.Context(), uc.Request{
			ActorRole: principal.Role,
			ActorID:   principal.UserID,
			ReportID:  reportID,
		})
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		out := make([]attachmentItem, 0, len(items))
		for _, a := range items {
			out = append(out, attachmentItem{
				ID:              a.ID,
				ReportID:        a.ReportID,
				UploadSessionID: a.UploadSessionID,
				FileName:        a.FileName,
				ContentType:     a.ContentType,
				FileSize:        a.FileSize,
				StorageKey:      a.StorageKey,
				DownloadURL:     a.SignedURL,
				CreatedAt:       a.CreatedAt.Unix(),
			})
		}

		shared.WriteJSON(w, http.StatusOK, responseBody{
			Items: out,
		})
	}
}
