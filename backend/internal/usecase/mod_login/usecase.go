package mod_login

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/user"
)

type Request struct {
	Email    string
	Password string
}

type Response struct {
	AccessToken    string
	RefreshTokenID string
	RefreshToken   string
}

type UseCase struct {
	users         UserFinder
	hasher        PasswordVerifier
	jwt           TokenIssuer
	refreshTokens RefreshTokenSaver
	random        RandomGenerator
	clock         Clock
	refreshTTL    time.Duration
}

func New(
	users UserFinder,
	hasher PasswordVerifier,
	jwt TokenIssuer,
	refreshTokens RefreshTokenSaver,
	random RandomGenerator,
	clock Clock,
	refreshTTL time.Duration,
) *UseCase {
	return &UseCase{
		users:         users,
		hasher:        hasher,
		jwt:           jwt,
		refreshTokens: refreshTokens,
		random:        random,
		clock:         clock,
		refreshTTL:    refreshTTL,
	}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) (Response, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	u, found, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return Response{}, err
	}
	if !found {
		return Response{}, domain.ErrInvalidCredentials
	}

	ok, err := uc.hasher.VerifyPassword(u.PasswordHash, req.Password)
	if err != nil {
		return Response{}, err
	}
	if !ok {
		return Response{}, domain.ErrInvalidCredentials
	}

	return uc.issueTokens(ctx, u.ID)
}

func (uc *UseCase) issueTokens(ctx context.Context, userID string) (Response, error) {
	access, err := uc.jwt.IssueAccessToken(userID)
	if err != nil {
		return Response{}, err
	}

	refreshSecret, err := uc.random.NewToken()
	if err != nil {
		return Response{}, err
	}
	now := uc.clock.Now()
	rt := user.RefreshToken{
		UserID:    userID,
		TokenHash: hashRefresh(refreshSecret),
		ExpiresAt: now.Add(uc.refreshTTL),
	}
	saved, err := uc.refreshTokens.Save(ctx, rt)
	if err != nil {
		return Response{}, err
	}

	return Response{
		AccessToken:    access,
		RefreshTokenID: saved.ID,
		RefreshToken:   refreshSecret,
	}, nil
}

func hashRefresh(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
