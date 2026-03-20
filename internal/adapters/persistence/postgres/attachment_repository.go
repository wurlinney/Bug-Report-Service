package postgres

import (
	"context"
	"errors"
	"hash/fnv"
	"strconv"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttachmentRepository struct {
	db *pgxpool.Pool
}

func parseAttachmentID(idText string) int64 {
	if i, err := strconv.ParseInt(idText, 10, 64); err == nil {
		return i
	}
	// Backward compatible fallback when DB still uses UUID for attachments.id.
	// We cannot map UUIDs to sequential numbers, but we can avoid runtime failures.
	h := fnv.New64a()
	_, _ = h.Write([]byte(idText))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

func NewAttachmentRepository(db *pgxpool.Pool) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

func (r *AttachmentRepository) Create(ctx context.Context, a ports.AttachmentRecord) (ports.AttachmentRecord, error) {
	const q = `
INSERT INTO attachments (
  bug_report_id, upload_session_id, file_name, content_type, file_size, storage_key, idempotency_key
 ) VALUES ($1::bigint,$2::bigint,$3,$4,$5,$6,$7)
RETURNING id::text, COALESCE(bug_report_id::text,''), COALESCE(upload_session_id::text,''), file_name, content_type, file_size, storage_key, created_at,
          COALESCE(idempotency_key,'')
`
	var idem any = nil
	var reportID any = nil
	var uploadSessionID any = nil
	if a.ReportID != "" {
		reportID = a.ReportID
	}
	if a.UploadSessionID != "" {
		uploadSessionID = a.UploadSessionID
	}
	if a.IdempotencyKey != "" {
		idem = a.IdempotencyKey
	}
	var created ports.AttachmentRecord
	var idText string
	err := r.db.QueryRow(ctx, q,
		reportID,
		uploadSessionID,
		a.FileName,
		a.ContentType,
		a.FileSize,
		a.StorageKey,
		idem,
	).Scan(
		&idText,
		&created.ReportID,
		&created.UploadSessionID,
		&created.FileName,
		&created.ContentType,
		&created.FileSize,
		&created.StorageKey,
		&created.CreatedAt,
		&created.IdempotencyKey,
	)
	if err != nil {
		return ports.AttachmentRecord{}, err
	}
	created.ID = parseAttachmentID(idText)
	return created, nil
}

func (r *AttachmentRepository) GetByIdempotencyKey(ctx context.Context, reportID string, uploadSessionID string, key string) (ports.AttachmentRecord, bool, error) {
	const q = `
SELECT id::text, COALESCE(bug_report_id::text,''), COALESCE(upload_session_id::text,''), file_name, content_type, file_size, storage_key, created_at,
       COALESCE(idempotency_key,'')
FROM attachments
WHERE bug_report_id IS NOT DISTINCT FROM NULLIF($1,'')::bigint
  AND upload_session_id IS NOT DISTINCT FROM NULLIF($2,'')::bigint
  AND idempotency_key = $3
`
	var a ports.AttachmentRecord
	var idText string
	err := r.db.QueryRow(ctx, q, reportID, uploadSessionID, key).Scan(
		&idText,
		&a.ReportID,
		&a.UploadSessionID,
		&a.FileName,
		&a.ContentType,
		&a.FileSize,
		&a.StorageKey,
		&a.CreatedAt,
		&a.IdempotencyKey,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.AttachmentRecord{}, false, nil
		}
		return ports.AttachmentRecord{}, false, err
	}
	a.ID = parseAttachmentID(idText)
	return a, true, nil
}

func (r *AttachmentRepository) ListByReport(ctx context.Context, reportID string) ([]ports.AttachmentRecord, error) {
	const q = `
SELECT id::text, bug_report_id::text, file_name, content_type, file_size, storage_key, created_at,
       COALESCE(idempotency_key,'')
FROM attachments
WHERE bug_report_id = $1::bigint
ORDER BY created_at
`
	rows, err := r.db.Query(ctx, q, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ports.AttachmentRecord
	for rows.Next() {
		var a ports.AttachmentRecord
		var idText string
		if err := rows.Scan(
			&idText,
			&a.ReportID,
			&a.FileName,
			&a.ContentType,
			&a.FileSize,
			&a.StorageKey,
			&a.CreatedAt,
			&a.IdempotencyKey,
		); err != nil {
			return nil, err
		}
		a.ID = parseAttachmentID(idText)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AttachmentRepository) ExistsByStorageKey(ctx context.Context, storageKey string) (bool, error) {
	const q = `SELECT 1 FROM attachments WHERE storage_key = $1 LIMIT 1`
	var one int
	err := r.db.QueryRow(ctx, q, storageKey).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *AttachmentRepository) BindSessionToReport(ctx context.Context, uploadSessionID string, reportID string) error {
	const q = `
UPDATE attachments
SET bug_report_id = $2,
    upload_session_id = NULL
WHERE upload_session_id = $1::bigint
`
	_, err := r.db.Exec(ctx, q, uploadSessionID, reportID)
	return err
}

func (r *AttachmentRepository) DeleteFromSessionByStorageKey(ctx context.Context, uploadSessionID string, storageKey string) (bool, error) {
	const q = `
DELETE FROM attachments
WHERE upload_session_id = $1::bigint
  AND storage_key = $2
`
	tag, err := r.db.Exec(ctx, q, uploadSessionID, storageKey)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
