package create_report

import (
	"encoding/json"
	"net/http"

	"bug-report-service/internal/api/shared"
	uc "bug-report-service/internal/usecase/create_report"
)

type requestBody struct {
	ReporterName    string `json:"reporter_name"`
	Description     string `json:"description"`
	UploadSessionID string `json:"upload_session_id"`
}

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
		var body requestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}

		result, err := useCase.Execute(r.Context(), uc.Request{
			ReporterName:    body.ReporterName,
			Description:     body.Description,
			UploadSessionID: body.UploadSessionID,
		})
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		shared.WriteJSON(w, http.StatusCreated, responseBody{
			ID:           result.ID,
			ReporterName: result.ReporterName,
			Description:  result.Description,
			Status:       result.Status,
			Influence:    result.Influence,
			Priority:     result.Priority,
			CreatedAt:    result.CreatedAt.Unix(),
			UpdatedAt:    result.UpdatedAt.Unix(),
		})
	}
}
