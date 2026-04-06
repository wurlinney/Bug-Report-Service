package list_notes

import (
	"net/http"

	"bug-report-service/internal/api/shared"
	uc "bug-report-service/internal/usecase/list_notes"

	"github.com/go-chi/chi/v5"
)

type noteItem struct {
	ID                string `json:"id"`
	ReportID          string `json:"report_id"`
	AuthorModeratorID string `json:"author_moderator_id"`
	Text              string `json:"text"`
	CreatedAt         int64  `json:"created_at"`
}

type responseBody struct {
	Items []noteItem `json:"items"`
	Total int        `json:"total"`
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

		q := r.URL.Query()

		items, total, err := useCase.Execute(r.Context(), uc.Request{
			ActorRole: principal.Role,
			ReportID:  reportID,
			Limit:     shared.ParseInt(q.Get("limit"), 20),
			Offset:    shared.ParseInt(q.Get("offset"), 0),
		})
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		out := make([]noteItem, 0, len(items))
		for _, n := range items {
			out = append(out, noteItem{
				ID:                n.ID,
				ReportID:          n.ReportID,
				AuthorModeratorID: n.AuthorModeratorID,
				Text:              n.Text,
				CreatedAt:         n.CreatedAt.Unix(),
			})
		}

		shared.WriteJSON(w, http.StatusOK, responseBody{
			Items: out,
			Total: total,
		})
	}
}
