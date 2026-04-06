package mod_login

import (
	"encoding/json"
	"net/http"

	"bug-report-service/internal/api/shared"
	uc "bug-report-service/internal/usecase/mod_login"
)

type requestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type responseBody struct {
	AccessToken    string `json:"access_token"`
	RefreshTokenID string `json:"refresh_token_id"`
	RefreshToken   string `json:"refresh_token"`
}

func New(useCase UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body requestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}

		result, err := useCase.Execute(r.Context(), uc.Request{
			Email:    body.Email,
			Password: body.Password,
		})
		if err != nil {
			shared.WriteDomainError(w, err)
			return
		}

		shared.WriteJSON(w, http.StatusOK, responseBody{
			AccessToken:    result.AccessToken,
			RefreshTokenID: result.RefreshTokenID,
			RefreshToken:   result.RefreshToken,
		})
	}
}
