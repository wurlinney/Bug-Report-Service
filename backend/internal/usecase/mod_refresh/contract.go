package mod_refresh

import (
	"context"
	"time"

	"bug-report-service/internal/domain/user"
)

type RefreshTokenGetter interface {
	GetActiveByID(ctx context.Context, id string) (user.RefreshToken, bool, error)
}

type RefreshTokenRevoker interface {
	Revoke(ctx context.Context, id string, when time.Time) error
}

type RefreshTokenSaver interface {
	Save(ctx context.Context, rt user.RefreshToken) (user.RefreshToken, error)
}

type TokenIssuer interface {
	IssueAccessToken(userID string) (string, error)
}

type RandomGenerator interface {
	NewToken() (string, error)
}

type Clock interface {
	Now() time.Time
}
