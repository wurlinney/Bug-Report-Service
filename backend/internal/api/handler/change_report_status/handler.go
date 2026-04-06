package change_report_status

import (
	"encoding/json"
	"net/http"

	"bug-report-service/internal/api/shared"
	ucMeta "bug-report-service/internal/usecase/change_report_meta"
	ucStatus "bug-report-service/internal/usecase/change_report_status"

	"github.com/go-chi/chi/v5"
)

type requestBody struct {
	Status    string `json:"status"`
	Priority  string `json:"priority,omitempty"`
	Influence string `json:"influence,omitempty"`
}

func New(statusUC StatusUseCase, metaUC MetaUseCase) http.HandlerFunc {
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

		if err := statusUC.Execute(r.Context(), ucStatus.Request{
			ActorRole: principal.Role,
			ReportID:  reportID,
			Status:    body.Status,
		}); err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		if body.Priority != "" && body.Influence != "" && metaUC != nil {
			if err := metaUC.Execute(r.Context(), ucMeta.Request{
				ActorRole: principal.Role,
				ReportID:  reportID,
				Priority:  body.Priority,
				Influence: body.Influence,
			}); err != nil {
				shared.WriteDomainError(w, err)
				return
			}
		}

		shared.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
