package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid = errors.New("token invalid")
	ErrTokenExpired = errors.New("token expired")
)

type JWTConfig struct {
	Issuer        string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	HMACSecretKey []byte
	Now           func() time.Time
}

type AccessClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

type JWTManager interface {
	IssueAccessToken(userID string) (string, error)
	VerifyAccessToken(token string) (AccessClaims, error)
}

type jwtManager struct {
	cfg JWTConfig
}

func NewJWTManager(cfg JWTConfig) JWTManager {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &jwtManager{cfg: cfg}
}

func (m *jwtManager) IssueAccessToken(userID string) (string, error) {
	now := m.cfg.Now()
	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.cfg.Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.cfg.AccessTTL)),
		},
		Role: "moderator",
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.cfg.HMACSecretKey)
}

func (m *jwtManager) VerifyAccessToken(tokenStr string) (AccessClaims, error) {
	var claims AccessClaims

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithTimeFunc(m.cfg.Now),
	)
	tok, err := parser.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (any, error) {
		return m.cfg.HMACSecretKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return AccessClaims{}, ErrTokenExpired
		}
		return AccessClaims{}, ErrTokenInvalid
	}
	if !tok.Valid {
		return AccessClaims{}, ErrTokenInvalid
	}
	if claims.Issuer != m.cfg.Issuer {
		return AccessClaims{}, ErrTokenInvalid
	}

	if claims.ExpiresAt != nil && m.cfg.Now().After(claims.ExpiresAt.Time) {
		return AccessClaims{}, ErrTokenExpired
	}

	return claims, nil
}
