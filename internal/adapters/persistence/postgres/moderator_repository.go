package postgres

import (
	"context"
	"errors"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ModeratorRepository struct {
	db *pgxpool.Pool
}

func NewModeratorRepository(db *pgxpool.Pool) *ModeratorRepository {
	return &ModeratorRepository{db: db}
}

func (r *ModeratorRepository) GetByEmail(ctx context.Context, email string) (ports.UserRecord, bool, error) {
	const q = `
SELECT id::text, name, email, password_hash, created_at, updated_at
FROM moderators
WHERE email = $1
`
	return r.getOne(ctx, q, email)
}

func (r *ModeratorRepository) GetByID(ctx context.Context, id string) (ports.UserRecord, bool, error) {
	const q = `
SELECT id::text, name, email, password_hash, created_at, updated_at
FROM moderators
WHERE id = $1::bigint
`
	return r.getOne(ctx, q, id)
}

func (r *ModeratorRepository) getOne(ctx context.Context, query string, arg string) (ports.UserRecord, bool, error) {
	var u ports.UserRecord
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
			return ports.UserRecord{}, false, nil
		}
		return ports.UserRecord{}, false, err
	}
	return u, true, nil
}

func (r *ModeratorRepository) Create(ctx context.Context, u ports.UserRecord) (ports.UserRecord, error) {
	const q = `
INSERT INTO moderators (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id::text, name, email, password_hash, created_at, updated_at
`
	var created ports.UserRecord
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
		if pgErr.Code == "23505" { // unique_violation
			return ports.UserRecord{}, ports.ErrUniqueViolation
		}
	}
	return ports.UserRecord{}, err
}

func (r *ModeratorRepository) UpsertByEmail(ctx context.Context, name string, email string, passwordHash string) error {
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
