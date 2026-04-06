package auth

import (
	"errors"

	"bug-report-service/internal/api/shared"
)

var errUnauthorized = errors.New("unauthorized")

// TokenVerifier adapts JWTManager to the api.TokenVerifier interface.
type TokenVerifier struct {
	JWT JWTManager
}

func (v TokenVerifier) VerifyAccessToken(token string) (shared.Principal, error) {
	claims, err := v.JWT.VerifyAccessToken(token)
	if err != nil {
		return shared.Principal{}, errUnauthorized
	}
	return shared.Principal{
		UserID: claims.Subject,
		Role:   claims.Role,
	}, nil
}
