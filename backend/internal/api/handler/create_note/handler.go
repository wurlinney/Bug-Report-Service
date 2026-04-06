package create_note

import (
	"encoding/json"
	"net/http"

	"bug-report-service/internal/api/shared"
	uc "bug-report-service/internal/usecase/create_note"

	"github.com/go-chi/chi/v5"
)

type requestBody struct {
	Text string `json:"text"`
}

type responseBody struct {
	ID                string `json:"id"`
	ReportID          string `json:"report_id"`
	AuthorModeratorID string `json:"author_moderator_id"`
	Text              string `json:"text"`
	CreatedAt         int64  `json:"created_at"`
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

		var body requestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}

		n, err := useCase.Execute(r.Context(), uc.Request{
			ActorRole: principal.Role,
			ActorID:   principal.UserID,
			ReportID:  reportID,
			Text:      body.Text,
		})
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		shared.WriteJSON(w, http.StatusCreated, responseBody{
			ID:                n.ID,
			ReportID:          n.ReportID,
			AuthorModeratorID: n.AuthorModeratorID,
			Text:              n.Text,
			CreatedAt:         n.CreatedAt.Unix(),
		})
	}
}
