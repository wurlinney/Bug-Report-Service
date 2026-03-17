package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"bug-report-service/internal/application/message"

	"github.com/go-chi/chi/v5"
)

type createMessageReq struct {
	Text string `json:"text"`
}

func createReportMessageHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if deps.MessageService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}

		var req createMessageReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		req.Text = strings.TrimSpace(req.Text)
		if req.Text == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "text is required")
			return
		}

		reportID := strings.TrimSpace(chi.URLParam(r, "id"))
		created, err := deps.MessageService.Create(r.Context(), message.CreateRequest{
			ActorRole: p.Role,
			ActorID:   p.UserID,
			ReportID:  reportID,
			Text:      req.Text,
		})
		if err != nil {
			switch err {
			case message.ErrBadInput:
				writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
			case message.ErrNotFound:
				writeError(w, http.StatusNotFound, "not_found", "not found")
			case message.ErrForbidden:
				writeError(w, http.StatusForbidden, "forbidden", "forbidden")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			}
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":          created.ID,
			"report_id":   created.ReportID,
			"sender_id":   created.SenderID,
			"sender_role": created.SenderRole,
			"text":        created.Text,
			"created_at":  created.CreatedAt.Unix(),
		})
	}
}

func listReportMessagesHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if deps.MessageService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}

		reportID := strings.TrimSpace(chi.URLParam(r, "id"))
		q := r.URL.Query()
		limit := parseInt(q.Get("limit"), 0)
		offset := parseInt(q.Get("offset"), 0)
		sortDesc := q.Get("sort_desc") == "1" || strings.EqualFold(q.Get("sort_desc"), "true")

		resp, err := deps.MessageService.List(r.Context(), message.ListRequest{
			ActorRole: p.Role,
			ActorID:   p.UserID,
			ReportID:  reportID,
			Limit:     limit,
			Offset:    offset,
			SortDesc:  sortDesc,
		})
		if err != nil {
			switch err {
			case message.ErrBadInput:
				writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
			case message.ErrNotFound:
				writeError(w, http.StatusNotFound, "not_found", "not found")
			case message.ErrForbidden:
				writeError(w, http.StatusForbidden, "forbidden", "forbidden")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			}
			return
		}

		out := make([]map[string]any, 0, len(resp.Items))
		for _, it := range resp.Items {
			out = append(out, map[string]any{
				"id":          it.ID,
				"report_id":   it.ReportID,
				"sender_id":   it.SenderID,
				"sender_role": it.SenderRole,
				"text":        it.Text,
				"created_at":  it.CreatedAt.Unix(),
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"items": out,
			"total": resp.Total,
		})
	}
}
