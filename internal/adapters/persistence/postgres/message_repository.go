package postgres

import (
	"context"
	"fmt"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, msg ports.MessageRecord) error {
	const q = `
INSERT INTO messages (id, bug_report_id, sender_id, sender_role, text, created_at)
VALUES ($1,$2,$3,$4,$5,$6)
`
	_, err := r.db.Exec(ctx, q, msg.ID, msg.ReportID, msg.SenderID, msg.SenderRole, msg.Text, msg.CreatedAt)
	return err
}

func (r *MessageRepository) ListByReport(ctx context.Context, reportID string, f ports.MessageListFilter) ([]ports.MessageRecord, int, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	dir := "ASC"
	if f.SortDesc {
		dir = "DESC"
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE bug_report_id = $1`, reportID).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf(`
SELECT id, bug_report_id, sender_id, sender_role, text, created_at
FROM messages
WHERE bug_report_id = $1
ORDER BY created_at %s
LIMIT $2 OFFSET $3
`, dir)
	rows, err := r.db.Query(ctx, q, reportID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ports.MessageRecord
	for rows.Next() {
		var m ports.MessageRecord
		if err := rows.Scan(&m.ID, &m.ReportID, &m.SenderID, &m.SenderRole, &m.Text, &m.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return out, total, nil
}
