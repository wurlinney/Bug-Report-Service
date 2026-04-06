package change_report_meta

import (
	"encoding/json"
	"net/http"

	"bug-report-service/internal/api/shared"
	uc "bug-report-service/internal/usecase/change_report_meta"

	"github.com/go-chi/chi/v5"
)

type requestBody struct {
	Priority  string `json:"priority"`
	Influence string `json:"influence"`
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

		if err := useCase.Execute(r.Context(), uc.Request{
			ActorRole: principal.Role,
			ReportID:  reportID,
			Priority:  body.Priority,
			Influence: body.Influence,
		}); err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		shared.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
