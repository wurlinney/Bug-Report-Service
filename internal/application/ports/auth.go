package ports

import (
	"context"
	"errors"
	"time"
)

var ErrUniqueViolation = errors.New("unique violation")

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (u UserRecord, found bool, err error)
	GetByID(ctx context.Context, id string) (u UserRecord, found bool, err error)
	Create(ctx context.Context, u UserRecord) error
}

type RefreshTokenRepository interface {
	Save(ctx context.Context, rt RefreshTokenRecord) error
	GetActiveByID(ctx context.Context, id string) (rt RefreshTokenRecord, found bool, err error)
	Revoke(ctx context.Context, id string, when time.Time) error
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	VerifyPassword(hash string, password string) (bool, error)
}

type AccessTokenIssuer interface {
	IssueAccessToken(userID string, role string) (string, error)
}

type Random interface {
	NewID() string
	NewToken() (string, error)
}

type Clock interface {
	Now() time.Time
}

type UserRecord struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RefreshTokenRecord struct {
	ID         string
	UserID     string
	Role       string
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *string
}
