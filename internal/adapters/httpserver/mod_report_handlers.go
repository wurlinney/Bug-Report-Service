package httpserver

import (
	"encoding/json"
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
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if deps.ReportService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
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
		var userIDPtr *string
		if s := strings.TrimSpace(qp.Get("user_id")); s != "" {
			userIDPtr = &s
		}

		from := parseUnixSeconds(qp.Get("created_from"))
		to := parseUnixSeconds(qp.Get("created_to"))

		items, total, err := deps.ReportService.ListAll(r.Context(), report.ListAllRequest{
			ActorRole: p.Role,
			Status:    statusPtr,
			UserID:    userIDPtr,
			Query:     queryPtr,
			CreatedFrom: func() *time.Time {
				if from.IsZero() {
					return nil
				}
				return &from
			}(),
			CreatedTo: func() *time.Time {
				if to.IsZero() {
					return nil
				}
				return &to
			}(),
			SortBy:   sortBy,
			SortDesc: sortDesc,
			Limit:    limit,
			Offset:   offset,
		})
		if err != nil {
			switch err {
			case report.ErrForbidden:
				writeError(w, http.StatusForbidden, "forbidden", "forbidden")
			case report.ErrBadInput:
				writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			}
			return
		}

		out := make([]map[string]any, 0, len(items))
		for _, it := range items {
			out = append(out, map[string]any{
				"id":          it.ID,
				"user_id":     it.UserID,
				"user_name":   it.UserName,
				"title":       it.Title,
				"description": it.Description,
				"status":      it.Status,
				"created_at":  it.CreatedAt.Unix(),
				"updated_at":  it.UpdatedAt.Unix(),
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
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if deps.ReportService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}

		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "id is required")
			return
		}

		got, err := deps.ReportService.GetForActor(r.Context(), p.Role, p.UserID, id)
		if err != nil {
			switch err {
			case report.ErrNotFound:
				writeError(w, http.StatusNotFound, "not_found", "not found")
			case report.ErrForbidden:
				writeError(w, http.StatusForbidden, "forbidden", "forbidden")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":          got.ID,
			"user_id":     got.UserID,
			"user_name":   got.UserName,
			"title":       got.Title,
			"description": got.Description,
			"status":      got.Status,
			"created_at":  got.CreatedAt.Unix(),
			"updated_at":  got.UpdatedAt.Unix(),
		})
	}
}

func changeReportStatusHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if deps.ReportService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
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
			switch err {
			case report.ErrBadInput:
				writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
			case report.ErrForbidden:
				writeError(w, http.StatusForbidden, "forbidden", "forbidden")
			case report.ErrNotFound:
				writeError(w, http.StatusNotFound, "not_found", "not found")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			}
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
