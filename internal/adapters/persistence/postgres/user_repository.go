package postgres

import (
	"context"
	"errors"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (ports.UserRecord, bool, error) {
	const q = `
SELECT id, email, password_hash, role, created_at, updated_at
FROM users
WHERE email = $1
`
	var u ports.UserRecord
	err := r.db.QueryRow(ctx, q, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.UserRecord{}, false, nil
		}
		return ports.UserRecord{}, false, err
	}
	return u, true, nil
}

func (r *UserRepository) Create(ctx context.Context, u ports.UserRecord) error {
	const q = `
INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
`
	_, err := r.db.Exec(ctx, q, u.ID, u.Email, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt)
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" { // unique_violation
			return ports.ErrUniqueViolation
		}
	}
	return err
}
