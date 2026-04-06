package mod_refresh

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/user"
)

type Request struct {
	RefreshTokenID string
	RefreshToken   string
}

type Response struct {
	AccessToken    string
	RefreshTokenID string
	RefreshToken   string
}

type UseCase struct {
	refreshTokens interface {
		RefreshTokenGetter
		RefreshTokenRevoker
		RefreshTokenSaver
	}
	jwt        TokenIssuer
	random     RandomGenerator
	clock      Clock
	refreshTTL time.Duration
}

func New(
	refreshTokens interface {
		RefreshTokenGetter
		RefreshTokenRevoker
		RefreshTokenSaver
	},
	jwt TokenIssuer,
	random RandomGenerator,
	clock Clock,
	refreshTTL time.Duration,
) *UseCase {
	return &UseCase{
		refreshTokens: refreshTokens,
		jwt:           jwt,
		random:        random,
		clock:         clock,
		refreshTTL:    refreshTTL,
	}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) (Response, error) {
	if req.RefreshTokenID == "" || req.RefreshToken == "" {
		return Response{}, domain.ErrInvalidRefresh
	}

	rt, found, err := uc.refreshTokens.GetActiveByID(ctx, req.RefreshTokenID)
	if err != nil {
		return Response{}, err
	}
	if !found {
		return Response{}, domain.ErrInvalidRefresh
	}

	now := uc.clock.Now()
	if !now.Before(rt.ExpiresAt) {
		_ = uc.refreshTokens.Revoke(ctx, rt.ID, now)
		return Response{}, domain.ErrInvalidRefresh
	}

	provided := hashRefresh(req.RefreshToken)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(rt.TokenHash)) != 1 {
		_ = uc.refreshTokens.Revoke(ctx, rt.ID, now)
		return Response{}, domain.ErrInvalidRefresh
	}

	_ = uc.refreshTokens.Revoke(ctx, rt.ID, now)
	return uc.issueTokens(ctx, rt.UserID)
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
