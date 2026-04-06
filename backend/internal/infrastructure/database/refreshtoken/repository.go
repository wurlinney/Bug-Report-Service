package refreshtoken

import (
	"context"
	"errors"
	"time"

	domainuser "bug-report-service/internal/domain/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, rt domainuser.RefreshToken) (domainuser.RefreshToken, error) {
	const q = `
INSERT INTO refresh_tokens (moderator_id, token_hash, expires_at, revoked_at, replaced_by)
VALUES ($1::bigint,$2,$3,$4,$5)
RETURNING id::text, moderator_id::text, token_hash, expires_at, created_at, revoked_at, replaced_by
`
	var saved domainuser.RefreshToken
	err := r.db.QueryRow(ctx, q,
		rt.UserID,
		rt.TokenHash,
		rt.ExpiresAt,
		rt.RevokedAt,
		rt.ReplacedBy,
	).Scan(
		&saved.ID,
		&saved.UserID,
		&saved.TokenHash,
		&saved.ExpiresAt,
		&saved.CreatedAt,
		&saved.RevokedAt,
		&saved.ReplacedBy,
	)
	if err != nil {
		return domainuser.RefreshToken{}, err
	}
	return saved, nil
}

func (r *Repository) GetActiveByID(ctx context.Context, id string) (domainuser.RefreshToken, bool, error) {
	const q = `
SELECT id::text, moderator_id::text, token_hash, expires_at, created_at, revoked_at, replaced_by
FROM refresh_tokens
WHERE id = $1::bigint AND revoked_at IS NULL
`
	var rt domainuser.RefreshToken
	err := r.db.QueryRow(ctx, q, id).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.CreatedAt,
		&rt.RevokedAt,
		&rt.ReplacedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domainuser.RefreshToken{}, false, nil
		}
		return domainuser.RefreshToken{}, false, err
	}
	return rt, true, nil
}

func (r *Repository) Revoke(ctx context.Context, id string, when time.Time) error {
	const q = `
UPDATE refresh_tokens
SET revoked_at = $2
WHERE id = $1::bigint AND revoked_at IS NULL
`
	_, err := r.db.Exec(ctx, q, id, when)
	return err
}
