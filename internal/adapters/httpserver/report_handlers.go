package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"bug-report-service/internal/application/report"

	"github.com/go-chi/chi/v5"
)

type createReportReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func createReportHandler(deps Deps) http.HandlerFunc {
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

		var req createReportReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		req.Description = strings.TrimSpace(req.Description)
		if req.Title == "" || req.Description == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "title and description are required")
			return
		}

		created, err := deps.ReportService.Create(r.Context(), report.CreateRequest{
			UserID:      p.UserID,
			Title:       req.Title,
			Description: req.Description,
		})
		if err != nil {
			if err == report.ErrBadInput {
				writeError(w, http.StatusBadRequest, "validation_error", "title and description are required")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":          created.ID,
			"user_id":     created.UserID,
			"title":       created.Title,
			"description": created.Description,
			"status":      created.Status,
			"created_at":  created.CreatedAt.Unix(),
			"updated_at":  created.UpdatedAt.Unix(),
		})
	}
}

func listMyReportsHandler(deps Deps) http.HandlerFunc {
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

		items, total, err := deps.ReportService.ListForUser(r.Context(), report.ListForUserRequest{
			ActorUserID: p.UserID,
			Status:      statusPtr,
			Query:       queryPtr,
			SortBy:      sortBy,
			SortDesc:    sortDesc,
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			if err == report.ErrBadInput {
				writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}

		out := make([]map[string]any, 0, len(items))
		for _, it := range items {
			out = append(out, map[string]any{
				"id":          it.ID,
				"user_id":     it.UserID,
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

func getMyReportHandler(deps Deps) http.HandlerFunc {
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

		got, err := deps.ReportService.GetForUser(r.Context(), p.UserID, id)
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
			"title":       got.Title,
			"description": got.Description,
			"status":      got.Status,
			"created_at":  got.CreatedAt.Unix(),
			"updated_at":  got.UpdatedAt.Unix(),
		})
	}
}

func parseInt(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
