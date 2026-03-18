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

func (r *NoteRepository) Create(ctx context.Context, n ports.InternalNoteRecord) error {
	const q = `
INSERT INTO internal_notes (id, bug_report_id, author_moderator_id, text, created_at)
VALUES ($1,$2,$3,$4,$5)
`
	_, err := r.db.Exec(ctx, q, n.ID, n.ReportID, n.AuthorModeratorID, n.Text, n.CreatedAt)
	return err
}

func (r *NoteRepository) ListByReport(ctx context.Context, reportID string, limit int, offset int) ([]ports.InternalNoteRecord, int, error) {
	const totalQ = `SELECT COUNT(*) FROM internal_notes WHERE bug_report_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, totalQ, reportID).Scan(&total); err != nil {
		return nil, 0, err
	}

	const listQ = `
SELECT id, bug_report_id, author_moderator_id, text, created_at
FROM internal_notes
WHERE bug_report_id = $1
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
