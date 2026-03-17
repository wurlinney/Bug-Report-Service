package security

import (
	"bug-report-service/internal/adapters/httpserver"
)

// TokenVerifier adapts JWTManager to httpserver.TokenVerifier.
type TokenVerifier struct {
	JWT JWTManager
}

func (v TokenVerifier) VerifyAccessToken(token string) (httpserver.Principal, error) {
	claims, err := v.JWT.VerifyAccessToken(token)
	if err != nil {
		return httpserver.Principal{}, httpserver.ErrUnauthorized
	}
	return httpserver.Principal{
		UserID: claims.Subject,
		Role:   claims.Role,
	}, nil
}
