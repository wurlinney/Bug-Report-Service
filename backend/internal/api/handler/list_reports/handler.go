package list_reports

import (
	"net/http"

	"bug-report-service/internal/api/shared"
	uc "bug-report-service/internal/usecase/list_reports"
)

type reportItem struct {
	ID           string `json:"id"`
	ReporterName string `json:"reporter_name"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	Influence    string `json:"influence,omitempty"`
	Priority     string `json:"priority,omitempty"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type responseBody struct {
	Items []reportItem `json:"items"`
	Total int          `json:"total"`
}

func New(useCase UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := shared.RequirePrincipal(w, r)
		if !ok {
			return
		}

		q := r.URL.Query()

		req := uc.Request{
			ActorRole: principal.Role,
			SortBy:    q.Get("sort_by"),
			SortDesc:  q.Get("sort_desc") == "true",
			Limit:     shared.ParseInt(q.Get("limit"), 20),
			Offset:    shared.ParseInt(q.Get("offset"), 0),
		}

		if v := q.Get("status"); v != "" {
			req.Status = &v
		}
		if v := q.Get("reporter_name"); v != "" {
			req.ReporterName = &v
		}
		if v := q.Get("q"); v != "" {
			req.Query = &v
		}
		if v := shared.ParseUnixSeconds(q.Get("created_from")); !v.IsZero() {
			req.CreatedFrom = shared.TimePtr(v)
		}
		if v := shared.ParseUnixSeconds(q.Get("created_to")); !v.IsZero() {
			req.CreatedTo = shared.TimePtr(v)
		}

		items, total, err := useCase.Execute(r.Context(), req)
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		out := make([]reportItem, 0, len(items))
		for _, rpt := range items {
			out = append(out, reportItem{
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

		shared.WriteJSON(w, http.StatusOK, responseBody{
			Items: out,
			Total: total,
		})
	}
}
