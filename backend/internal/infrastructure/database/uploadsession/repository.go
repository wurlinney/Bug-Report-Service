package uploadsession

import (
	"context"
	"errors"

	domainuploadsession "bug-report-service/internal/domain/uploadsession"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context) (domainuploadsession.UploadSession, error) {
	const q = `
INSERT INTO upload_sessions DEFAULT VALUES
RETURNING id::text, created_at
`
	var rec domainuploadsession.UploadSession
	if err := r.db.QueryRow(ctx, q).Scan(&rec.ID, &rec.CreatedAt); err != nil {
		return domainuploadsession.UploadSession{}, err
	}
	return rec, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domainuploadsession.UploadSession, bool, error) {
	const q = `
SELECT id::text, created_at
FROM upload_sessions
WHERE id = $1::bigint
`
	var rec domainuploadsession.UploadSession
	if err := r.db.QueryRow(ctx, q, id).Scan(&rec.ID, &rec.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domainuploadsession.UploadSession{}, false, nil
		}
		return domainuploadsession.UploadSession{}, false, err
	}
	return rec, true, nil
}
