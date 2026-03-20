package postgres

import (
	"context"
	"errors"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UploadSessionRepository struct {
	db *pgxpool.Pool
}

func NewUploadSessionRepository(db *pgxpool.Pool) *UploadSessionRepository {
	return &UploadSessionRepository{db: db}
}

func (r *UploadSessionRepository) Create(ctx context.Context) (ports.UploadSessionRecord, error) {
	const q = `
INSERT INTO upload_sessions DEFAULT VALUES
RETURNING id::text, created_at
`
	var rec ports.UploadSessionRecord
	if err := r.db.QueryRow(ctx, q).Scan(&rec.ID, &rec.CreatedAt); err != nil {
		return ports.UploadSessionRecord{}, err
	}
	return rec, nil
}

func (r *UploadSessionRepository) GetByID(ctx context.Context, id string) (ports.UploadSessionRecord, bool, error) {
	const q = `
SELECT id::text, created_at
FROM upload_sessions
WHERE id = $1::bigint
`
	var rec ports.UploadSessionRecord
	if err := r.db.QueryRow(ctx, q, id).Scan(&rec.ID, &rec.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.UploadSessionRecord{}, false, nil
		}
		return ports.UploadSessionRecord{}, false, err
	}
	return rec, true, nil
}
