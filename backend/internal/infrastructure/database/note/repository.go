package note

import (
	"context"

	domainnote "bug-report-service/internal/domain/note"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, n domainnote.Note) (domainnote.Note, error) {
	const q = `
INSERT INTO internal_notes (bug_report_id, author_moderator_id, text)
VALUES ($1::bigint,$2::bigint,$3)
RETURNING id::text, bug_report_id::text, author_moderator_id::text, text, created_at
`
	var created domainnote.Note
	err := r.db.QueryRow(ctx, q, n.ReportID, n.AuthorModeratorID, n.Text).Scan(
		&created.ID,
		&created.ReportID,
		&created.AuthorModeratorID,
		&created.Text,
		&created.CreatedAt,
	)
	if err != nil {
		return domainnote.Note{}, err
	}
	return created, nil
}

func (r *Repository) ListByReport(ctx context.Context, reportID string, limit int, offset int) ([]domainnote.Note, int, error) {
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

	out := make([]domainnote.Note, 0, limit)
	for rows.Next() {
		var n domainnote.Note
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
