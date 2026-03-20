package postgres

import (
	"context"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NoteRepository struct {
	db *pgxpool.Pool
}

func NewNoteRepository(db *pgxpool.Pool) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) Create(ctx context.Context, n ports.InternalNoteRecord) (ports.InternalNoteRecord, error) {
	const q = `
INSERT INTO internal_notes (bug_report_id, author_moderator_id, text)
VALUES ($1::bigint,$2::bigint,$3)
RETURNING id::text, bug_report_id::text, author_moderator_id::text, text, created_at
`
	var created ports.InternalNoteRecord
	err := r.db.QueryRow(ctx, q, n.ReportID, n.AuthorModeratorID, n.Text).Scan(
		&created.ID,
		&created.ReportID,
		&created.AuthorModeratorID,
		&created.Text,
		&created.CreatedAt,
	)
	if err != nil {
		return ports.InternalNoteRecord{}, err
	}
	return created, nil
}

func (r *NoteRepository) ListByReport(ctx context.Context, reportID string, limit int, offset int) ([]ports.InternalNoteRecord, int, error) {
	const totalQ = `SELECT COUNT(*) FROM internal_notes WHERE bug_report_id = $1::bigint`
	var total int
	if err := r.db.QueryRow(ctx, totalQ, reportID).Scan(&total); err != nil {
		return nil, 0, err
	}

	const listQ = `
SELECT id::text, bug_report_id::text, author_moderator_id::text, text, created_at
FROM internal_notes
WHERE bug_report_id = $1::bigint
ORDER BY created_at
LIMIT $2 OFFSET $3
`
	rows, err := r.db.Query(ctx, listQ, reportID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]ports.InternalNoteRecord, 0, limit)
	for rows.Next() {
		var n ports.InternalNoteRecord
		if err := rows.Scan(&n.ID, &n.ReportID, &n.AuthorModeratorID, &n.Text, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
