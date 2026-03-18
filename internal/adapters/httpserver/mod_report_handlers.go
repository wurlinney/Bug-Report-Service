package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bug-report-service/internal/application/report"

	"github.com/go-chi/chi/v5"
)

type changeStatusReq struct {
	Status string `json:"status"`
}

func listAllReportsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := requirePrincipal(w, r)
		if !ok || deps.ReportService == nil {
			if ok {
				writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			}
			return
		}

		qp := r.URL.Query()
		limit := parseInt(qp.Get("limit"), 0)
		offset := parseInt(qp.Get("offset"), 0)
		sortBy := strings.TrimSpace(qp.Get("sort_by"))
		sortDesc := qp.Get("sort_desc") == "1" || strings.EqualFold(qp.Get("sort_desc"), "true")

		var statusPtr *string
		if s := strings.TrimSpace(qp.Get("status")); s != "" {
			statusPtr = &s
		}
		var queryPtr *string
		if s := strings.TrimSpace(qp.Get("q")); s != "" {
			queryPtr = &s
		}
		var reporterNamePtr *string
		if s := strings.TrimSpace(qp.Get("reporter_name")); s != "" {
			reporterNamePtr = &s
		}

		from := parseUnixSeconds(qp.Get("created_from"))
		to := parseUnixSeconds(qp.Get("created_to"))

		items, total, err := deps.ReportService.ListAll(r.Context(), report.ListAllRequest{
			ActorRole:    p.Role,
			Status:       statusPtr,
			ReporterName: reporterNamePtr,
			Query:        queryPtr,
			CreatedFrom:  timePtr(from),
			CreatedTo:    timePtr(to),
			SortBy:       sortBy,
			SortDesc:     sortDesc,
			Limit:        limit,
			Offset:       offset,
		})
		if err != nil {
			writeReportServiceError(w, err)
			return
		}

		out := make([]map[string]any, 0, len(items))
		for _, it := range items {
			out = append(out, map[string]any{
				"id":            it.ID,
				"reporter_name": it.ReporterName,
				"description":   it.Description,
				"status":        it.Status,
				"created_at":    it.CreatedAt.Unix(),
				"updated_at":    it.UpdatedAt.Unix(),
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"items": out,
			"total": total,
		})
	}
}

func getReportHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := requirePrincipal(w, r)
		if !ok || deps.ReportService == nil {
			if ok {
				writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			}
			return
		}

		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "id is required")
			return
		}

		got, err := deps.ReportService.GetForActor(r.Context(), p.Role, p.UserID, id)
		if err != nil {
			writeReportServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":            got.ID,
			"reporter_name": got.ReporterName,
			"description":   got.Description,
			"status":        got.Status,
			"created_at":    got.CreatedAt.Unix(),
			"updated_at":    got.UpdatedAt.Unix(),
		})
	}
}

func changeReportStatusHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := requirePrincipal(w, r)
		if !ok || deps.ReportService == nil {
			if ok {
				writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			}
			return
		}

		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "id is required")
			return
		}

		var req changeStatusReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		req.Status = strings.TrimSpace(req.Status)

		if err := deps.ReportService.ChangeStatus(r.Context(), report.ChangeStatusRequest{
			ActorRole: p.Role,
			ReportID:  id,
			Status:    req.Status,
		}); err != nil {
			writeReportServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func parseUnixSeconds(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

func timePtr(v time.Time) *time.Time {
	if v.IsZero() {
		return nil
	}
	return &v
}

func writeReportServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, report.ErrBadInput):
		writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
	case errors.Is(err, report.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, report.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "forbidden")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}
