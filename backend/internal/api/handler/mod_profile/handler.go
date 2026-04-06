package mod_profile

import (
	"net/http"

	"bug-report-service/internal/api/shared"
)

type responseBody struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func New(useCase UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := shared.RequirePrincipal(w, r)
		if !ok {
			return
		}

		profile, err := useCase.Execute(r.Context(), principal.UserID)
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		shared.WriteJSON(w, http.StatusOK, responseBody{
			ID:        profile.ID,
			Name:      profile.Name,
			Email:     profile.Email,
			CreatedAt: profile.CreatedAt,
			UpdatedAt: profile.UpdatedAt,
		})
	}
}
