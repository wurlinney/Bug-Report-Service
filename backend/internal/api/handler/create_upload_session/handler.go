package create_upload_session

import (
	"net/http"

	"bug-report-service/internal/api/shared"
)

type responseBody struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"created_at"`
}

func New(useCase UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := useCase.Execute(r.Context())
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		shared.WriteJSON(w, http.StatusCreated, responseBody{
			ID:        session.ID,
			CreatedAt: session.CreatedAt.Unix(),
		})
	}
}
