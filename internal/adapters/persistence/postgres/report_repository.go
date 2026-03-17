package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, rep ports.ReportRecord) error {
	const q = `
INSERT INTO bug_reports (id, user_id, title, description, status, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
`
	_, err := r.db.Exec(ctx, q, rep.ID, rep.UserID, rep.Title, rep.Description, rep.Status, rep.CreatedAt, rep.UpdatedAt)
	return err
}

func (r *ReportRepository) GetByID(ctx context.Context, id string) (ports.ReportRecord, bool, error) {
	const q = `
SELECT id, user_id, title, description, status, created_at, updated_at
FROM bug_reports
WHERE id = $1
`
	var rep ports.ReportRecord
	err := r.db.QueryRow(ctx, q, id).Scan(
		&rep.ID,
		&rep.UserID,
		&rep.Title,
		&rep.Description,
		&rep.Status,
		&rep.CreatedAt,
		&rep.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ReportRecord{}, false, nil
		}
		return ports.ReportRecord{}, false, err
	}
	return rep, true, nil
}

func (r *ReportRepository) UpdateStatus(ctx context.Context, id string, status string, updatedAt time.Time) error {
	const q = `
UPDATE bug_reports
SET status = $2, updated_at = $3
WHERE id = $1
`
	ct, err := r.db.Exec(ctx, q, id, status, updatedAt)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *ReportRepository) ListByUser(ctx context.Context, userID string, f ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	f2 := f
	f2.UserID = &userID
	return r.list(ctx, f2)
}

func (r *ReportRepository) ListAll(ctx context.Context, f ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	return r.list(ctx, f)
}

func (r *ReportRepository) list(ctx context.Context, f ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	where, args := buildReportWhere(f)

	sortCol := "created_at"
	if f.SortBy == "updated_at" {
		sortCol = "updated_at"
	}
	dir := "ASC"
	if f.SortDesc {
		dir = "DESC"
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	// total
	totalQ := "SELECT COUNT(*) FROM bug_reports" + where
	var total int
	if err := r.db.QueryRow(ctx, totalQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// list
	args2 := append(append([]any{}, args...), limit, offset)
	listQ := fmt.Sprintf(`
SELECT id, user_id, title, description, status, created_at, updated_at
FROM bug_reports
%s
ORDER BY %s %s
LIMIT $%d OFFSET $%d
`, where, sortCol, dir, len(args)+1, len(args)+2)

	rows, err := r.db.Query(ctx, listQ, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ports.ReportRecord
	for rows.Next() {
		var rep ports.ReportRecord
		if err := rows.Scan(&rep.ID, &rep.UserID, &rep.Title, &rep.Description, &rep.Status, &rep.CreatedAt, &rep.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, rep)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return out, total, nil
}

func buildReportWhere(f ports.ReportListFilter) (string, []any) {
	var parts []string
	var args []any
	addArg := func(val any) string {
		args = append(args, val)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.Status != nil && strings.TrimSpace(*f.Status) != "" {
		parts = append(parts, "status = "+addArg(strings.TrimSpace(*f.Status)))
	}
	if f.UserID != nil && strings.TrimSpace(*f.UserID) != "" {
		parts = append(parts, "user_id = "+addArg(strings.TrimSpace(*f.UserID)))
	}
	if f.Query != nil && strings.TrimSpace(*f.Query) != "" {
		q := "%" + strings.TrimSpace(*f.Query) + "%"
		p1 := addArg(q)
		p2 := addArg(q)
		parts = append(parts, "(title ILIKE "+p1+" OR description ILIKE "+p2+")")
	}
	if f.CreatedAt != nil {
		if f.CreatedAt.From != nil {
			parts = append(parts, "created_at >= "+addArg(*f.CreatedAt.From))
		}
		if f.CreatedAt.To != nil {
			parts = append(parts, "created_at <= "+addArg(*f.CreatedAt.To))
		}
	}

	if len(parts) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}
