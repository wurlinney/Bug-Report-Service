package mod_login

import (
	"context"
	"time"

	"bug-report-service/internal/domain/user"
)

type UserFinder interface {
	GetByEmail(ctx context.Context, email string) (user.User, bool, error)
}

type PasswordVerifier interface {
	VerifyPassword(hash string, password string) (bool, error)
}

type TokenIssuer interface {
	IssueAccessToken(userID string) (string, error)
}

type RefreshTokenSaver interface {
	Save(ctx context.Context, rt user.RefreshToken) (user.RefreshToken, error)
}

type RandomGenerator interface {
	NewToken() (string, error)
}

type Clock interface {
	Now() time.Time
}
