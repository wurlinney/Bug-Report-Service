package postgres

import (
	"context"
	"errors"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttachmentRepository struct {
	db *pgxpool.Pool
}

func NewAttachmentRepository(db *pgxpool.Pool) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

func (r *AttachmentRepository) Create(ctx context.Context, a ports.AttachmentRecord) error {
	const q = `
INSERT INTO attachments (
  id, bug_report_id, file_name, content_type, file_size, storage_key, created_at,
  idempotency_key, uploaded_by_id, uploaded_by_role
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
`
	var idem any = nil
	if a.IdempotencyKey != "" {
		idem = a.IdempotencyKey
	}
	_, err := r.db.Exec(ctx, q,
		a.ID,
		a.ReportID,
		a.FileName,
		a.ContentType,
		a.FileSize,
		a.StorageKey,
		a.CreatedAt,
		idem,
		a.UploadedByID,
		a.UploadedByRole,
	)
	return err
}

func (r *AttachmentRepository) GetByIdempotencyKey(ctx context.Context, reportID string, key string) (ports.AttachmentRecord, bool, error) {
	const q = `
SELECT id, bug_report_id, file_name, content_type, file_size, storage_key, created_at,
       COALESCE(idempotency_key,''), uploaded_by_id, uploaded_by_role
FROM attachments
WHERE bug_report_id = $1 AND idempotency_key = $2
`
	var a ports.AttachmentRecord
	err := r.db.QueryRow(ctx, q, reportID, key).Scan(
		&a.ID,
		&a.ReportID,
		&a.FileName,
		&a.ContentType,
		&a.FileSize,
		&a.StorageKey,
		&a.CreatedAt,
		&a.IdempotencyKey,
		&a.UploadedByID,
		&a.UploadedByRole,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.AttachmentRecord{}, false, nil
		}
		return ports.AttachmentRecord{}, false, err
	}
	return a, true, nil
}

func (r *AttachmentRepository) ListByReport(ctx context.Context, reportID string) ([]ports.AttachmentRecord, error) {
	const q = `
SELECT id, bug_report_id, file_name, content_type, file_size, storage_key, created_at,
       COALESCE(idempotency_key,''), uploaded_by_id, uploaded_by_role
FROM attachments
WHERE bug_report_id = $1
ORDER BY created_at ASC
`
	rows, err := r.db.Query(ctx, q, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ports.AttachmentRecord
	for rows.Next() {
		var a ports.AttachmentRecord
		if err := rows.Scan(
			&a.ID,
			&a.ReportID,
			&a.FileName,
			&a.ContentType,
			&a.FileSize,
			&a.StorageKey,
			&a.CreatedAt,
			&a.IdempotencyKey,
			&a.UploadedByID,
			&a.UploadedByRole,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
