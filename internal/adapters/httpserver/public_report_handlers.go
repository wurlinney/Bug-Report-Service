package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bug-report-service/internal/application/report"
)

type createPublicReportReq struct {
	ReporterName string `json:"reporter_name"`
	Description  string `json:"description"`
}

func createPublicReportHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.ReportService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}

		var req createPublicReportReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		req.ReporterName = strings.TrimSpace(req.ReporterName)
		req.Description = strings.TrimSpace(req.Description)
		if req.ReporterName == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "reporter_name is required")
			return
		}

		created, err := deps.ReportService.Create(r.Context(), report.CreateRequest{
			ReporterName: req.ReporterName,
			Description:  req.Description,
		})
		if err != nil {
			if errors.Is(err, report.ErrBadInput) {
				writeError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":            created.ID,
			"reporter_name": created.ReporterName,
			"description":   created.Description,
			"status":        created.Status,
			"created_at":    created.CreatedAt.Unix(),
			"updated_at":    created.UpdatedAt.Unix(),
			"message":       "Ваше обращение принято в работу",
		})
	}
}
