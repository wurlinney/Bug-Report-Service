package get_report

import (
	"net/http"

	"bug-report-service/internal/api/shared"

	"github.com/go-chi/chi/v5"
)

type responseBody struct {
	ID           string `json:"id"`
	ReporterName string `json:"reporter_name"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	Influence    string `json:"influence,omitempty"`
	Priority     string `json:"priority,omitempty"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
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

		rpt, err := useCase.Execute(r.Context(), principal.Role, reportID)
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		shared.WriteJSON(w, http.StatusOK, responseBody{
			ID:           rpt.ID,
			ReporterName: rpt.ReporterName,
			Description:  rpt.Description,
			Status:       rpt.Status,
			Influence:    rpt.Influence,
			Priority:     rpt.Priority,
			CreatedAt:    rpt.CreatedAt.Unix(),
			UpdatedAt:    rpt.UpdatedAt.Unix(),
		})
	}
}
