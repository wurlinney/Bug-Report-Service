package postgres

import (
	"context"
	"errors"
	"time"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, rt ports.RefreshTokenRecord) (ports.RefreshTokenRecord, error) {
	const q = `
INSERT INTO refresh_tokens (moderator_id, token_hash, expires_at, revoked_at, replaced_by)
VALUES ($1::bigint,$2,$3,$4,$5)
RETURNING id::text, moderator_id::text, token_hash, expires_at, created_at, revoked_at, replaced_by
`
	var saved ports.RefreshTokenRecord
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
		return ports.RefreshTokenRecord{}, err
	}
	return saved, nil
}

func (r *RefreshTokenRepository) GetActiveByID(ctx context.Context, id string) (ports.RefreshTokenRecord, bool, error) {
	const q = `
SELECT id::text, moderator_id::text, token_hash, expires_at, created_at, revoked_at, replaced_by
FROM refresh_tokens
WHERE id = $1::bigint AND revoked_at IS NULL
`
	var rt ports.RefreshTokenRecord
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
			return ports.RefreshTokenRecord{}, false, nil
		}
		return ports.RefreshTokenRecord{}, false, err
	}
	return rt, true, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string, when time.Time) error {
	const q = `
UPDATE refresh_tokens
SET revoked_at = $2
WHERE id = $1::bigint AND revoked_at IS NULL
`
	_, err := r.db.Exec(ctx, q, id, when)
	return err
}
