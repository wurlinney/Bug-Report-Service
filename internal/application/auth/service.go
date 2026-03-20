package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"bug-report-service/internal/application/ports"

	"crypto/subtle"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

type Deps struct {
	Users         ports.UserRepository
	RefreshTokens ports.RefreshTokenRepository
	Hasher        ports.PasswordHasher
	JWT           ports.AccessTokenIssuer
	Random        ports.Random
	Clock         ports.Clock

	RefreshTTL time.Duration
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	email := normalizeEmail(req.Email)
	u, found, err := s.deps.Users.GetByEmail(ctx, email)
	if err != nil {
		return AuthResponse{}, err
	}
	if !found {
		return AuthResponse{}, ErrInvalidCredentials
	}

	ok, err := s.deps.Hasher.VerifyPassword(u.PasswordHash, req.Password)
	if err != nil {
		return AuthResponse{}, err
	}
	if !ok {
		return AuthResponse{}, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, u.ID)
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (AuthResponse, error) {
	if req.RefreshTokenID == "" || req.RefreshToken == "" {
		return AuthResponse{}, ErrInvalidRefreshToken
	}

	rt, found, err := s.deps.RefreshTokens.GetActiveByID(ctx, req.RefreshTokenID)
	if err != nil {
		return AuthResponse{}, err
	}
	if !found {
		return AuthResponse{}, ErrInvalidRefreshToken
	}

	now := s.deps.Clock.Now()
	if !now.Before(rt.ExpiresAt) {
		_ = s.deps.RefreshTokens.Revoke(ctx, rt.ID, now)
		return AuthResponse{}, ErrInvalidRefreshToken
	}

	provided := hashRefresh(req.RefreshToken)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(rt.TokenHash)) != 1 {
		_ = s.deps.RefreshTokens.Revoke(ctx, rt.ID, now)
		return AuthResponse{}, ErrInvalidRefreshToken
	}

	// rotate: revoke old, issue new
	_ = s.deps.RefreshTokens.Revoke(ctx, rt.ID, now)
	return s.issueTokens(ctx, rt.UserID)
}

func (s *Service) issueTokens(ctx context.Context, userID string) (AuthResponse, error) {
	access, err := s.deps.JWT.IssueAccessToken(userID)
	if err != nil {
		return AuthResponse{}, err
	}

	refreshSecret, err := s.deps.Random.NewToken()
	if err != nil {
		return AuthResponse{}, err
	}
	now := s.deps.Clock.Now()
	rt := ports.RefreshTokenRecord{
		UserID:    userID,
		TokenHash: hashRefresh(refreshSecret),
		ExpiresAt: now.Add(s.deps.RefreshTTL),
	}
	saved, err := s.deps.RefreshTokens.Save(ctx, rt)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		AccessToken:    access,
		RefreshTokenID: saved.ID,
		RefreshToken:   refreshSecret,
	}, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// hashRefresh stores refresh token as hash; the plain token is only returned once.
func hashRefresh(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
