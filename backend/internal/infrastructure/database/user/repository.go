package user

import (
	"context"
	"errors"

	"bug-report-service/internal/domain"
	domainuser "bug-report-service/internal/domain/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (domainuser.User, bool, error) {
	const q = `
SELECT id::text, name, email, password_hash, created_at, updated_at
FROM moderators
WHERE email = $1
`
	return r.getOne(ctx, q, email)
}

func (r *Repository) GetByID(ctx context.Context, id string) (domainuser.User, bool, error) {
	const q = `
SELECT id::text, name, email, password_hash, created_at, updated_at
FROM moderators
WHERE id = $1::bigint
`
	return r.getOne(ctx, q, id)
}

func (r *Repository) getOne(ctx context.Context, query string, arg string) (domainuser.User, bool, error) {
	var u domainuser.User
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domainuser.User{}, false, nil
		}
		return domainuser.User{}, false, err
	}
	return u, true, nil
}

func (r *Repository) Create(ctx context.Context, u domainuser.User) (domainuser.User, error) {
	const q = `
INSERT INTO moderators (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id::text, name, email, password_hash, created_at, updated_at
`
	var created domainuser.User
	err := r.db.QueryRow(ctx, q, u.Name, u.Email, u.PasswordHash).Scan(
		&created.ID,
		&created.Name,
		&created.Email,
		&created.PasswordHash,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err == nil {
		return created, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return domainuser.User{}, domain.ErrUniqueViolation
		}
	}
	return domainuser.User{}, err
}

func (r *Repository) UpsertByEmail(ctx context.Context, name string, email string, passwordHash string) error {
	const q = `
INSERT INTO moderators (name, email, password_hash)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET
  name = EXCLUDED.name,
  password_hash = EXCLUDED.password_hash,
  updated_at = NOW()
`
	_, err := r.db.Exec(ctx, q, name, email, passwordHash)
	return err
}
