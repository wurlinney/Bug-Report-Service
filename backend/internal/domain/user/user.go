package user

import "time"

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RefreshToken struct {
	ID         string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *string
}
