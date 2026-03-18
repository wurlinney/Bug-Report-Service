package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bug-report-service/internal/application/note"

	"github.com/go-chi/chi/v5"
)

type createNoteReq struct {
	Text string `json:"text"`
}

func createReportNoteHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := requirePrincipal(w, r)
		if !ok || deps.NoteService == nil {
			if ok {
				writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			}
			return
		}
		reportID := strings.TrimSpace(chi.URLParam(r, "id"))
		var req createNoteReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		req.Text = strings.TrimSpace(req.Text)
		created, err := deps.NoteService.Create(r.Context(), note.CreateRequest{
			ActorRole: p.Role,
			ActorID:   p.UserID,
			ReportID:  reportID,
			Text:      req.Text,
		})
		if err != nil {
			writeNoteServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":                  created.ID,
			"report_id":           created.ReportID,
			"author_moderator_id": created.AuthorModeratorID,
			"text":                created.Text,
			"created_at":          created.CreatedAt,
		})
	}
}

func listReportNotesHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := requirePrincipal(w, r)
		if !ok || deps.NoteService == nil {
			if ok {
				writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			}
			return
		}
		reportID := strings.TrimSpace(chi.URLParam(r, "id"))
		limit := parseInt(r.URL.Query().Get("limit"), 20)
		offset := parseInt(r.URL.Query().Get("offset"), 0)
		items, total, err := deps.NoteService.List(r.Context(), note.ListRequest{
			ActorRole: p.Role,
			ReportID:  reportID,
			Limit:     limit,
			Offset:    offset,
		})
		if err != nil {
			writeNoteServiceError(w, err)
			return
		}
		out := make([]map[string]any, 0, len(items))
		for _, it := range items {
			out = append(out, map[string]any{
				"id":                  it.ID,
				"report_id":           it.ReportID,
				"author_moderator_id": it.AuthorModeratorID,
				"text":                it.Text,
				"created_at":          it.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out, "total": total})
	}
}

func writeNoteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, note.ErrBadInput):
		writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
	case errors.Is(err, note.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, note.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "forbidden")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}
