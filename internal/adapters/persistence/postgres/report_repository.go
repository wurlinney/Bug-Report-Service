package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bug-report-service/internal/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

type reportScanner interface {
	Scan(dest ...any) error
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, rep ports.ReportRecord) (ports.ReportRecord, error) {
	const q = `
INSERT INTO bug_reports (reporter_name, description, status)
VALUES ($1,$2,$3)
RETURNING id::text, reporter_name, description, status, influence, priority, created_at, updated_at
`
	created, err := scanReport(r.db.QueryRow(ctx, q, rep.ReporterName, rep.Description, rep.Status))
	if err != nil {
		return ports.ReportRecord{}, err
	}
	return created, nil
}

func (r *ReportRepository) GetByID(ctx context.Context, id string) (ports.ReportRecord, bool, error) {
	const q = `
SELECT r.id::text, r.reporter_name, r.description, r.status, r.influence, r.priority, r.created_at, r.updated_at
FROM bug_reports r
WHERE r.id = $1::bigint
`
	rep, err := scanReport(r.db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ReportRecord{}, false, nil
		}
		return ports.ReportRecord{}, false, err
	}
	return rep, true, nil
}

func (r *ReportRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	const q = `
UPDATE bug_reports
SET status = $2
WHERE id = $1::bigint
`
	ct, err := r.db.Exec(ctx, q, id, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *ReportRepository) UpdateMeta(ctx context.Context, id string, priority string, influence string) error {
	const q = `
UPDATE bug_reports
SET priority = $2,
    influence = $3
WHERE id = $1::bigint
`
	ct, err := r.db.Exec(ctx, q, id, priority, influence)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *ReportRepository) ListAll(ctx context.Context, f ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	return r.list(ctx, f)
}

func (r *ReportRepository) list(ctx context.Context, f ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	where, args := buildReportWhere(f)

	sortCol := "r.created_at"
	if f.SortBy == "updated_at" {
		sortCol = "r.updated_at"
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
	totalQ := "SELECT COUNT(*) FROM bug_reports r" + where
	var total int
	if err := r.db.QueryRow(ctx, totalQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// list
	args2 := append(append([]any{}, args...), limit, offset)
	listQ := `
SELECT r.id::text, r.reporter_name, r.description, r.status, r.influence, r.priority, r.created_at, r.updated_at
FROM bug_reports r` + where + `
ORDER BY ` + sortCol + ` ` + dir + `
LIMIT ` + fmt.Sprintf("$%d", len(args)+1) + ` OFFSET ` + fmt.Sprintf("$%d", len(args)+2)

	rows, err := r.db.Query(ctx, listQ, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ports.ReportRecord
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
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
		parts = append(parts, "r.status = "+addArg(strings.TrimSpace(*f.Status)))
	}
	if f.ReporterName != nil && strings.TrimSpace(*f.ReporterName) != "" {
		parts = append(parts, "r.reporter_name ILIKE "+addArg("%"+strings.TrimSpace(*f.ReporterName)+"%"))
	}
	if f.Query != nil && strings.TrimSpace(*f.Query) != "" {
		q := "%" + strings.TrimSpace(*f.Query) + "%"
		p1 := addArg(q)
		p2 := addArg(q)
		parts = append(parts, "(r.reporter_name ILIKE "+p1+" OR r.description ILIKE "+p2+")")
	}
	if f.CreatedAt != nil {
		if f.CreatedAt.From != nil {
			parts = append(parts, "r.created_at >= "+addArg(*f.CreatedAt.From))
		}
		if f.CreatedAt.To != nil {
			parts = append(parts, "r.created_at <= "+addArg(*f.CreatedAt.To))
		}
	}

	if len(parts) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func scanReport(s reportScanner) (ports.ReportRecord, error) {
	var rep ports.ReportRecord
	err := s.Scan(
		&rep.ID,
		&rep.ReporterName,
		&rep.Description,
		&rep.Status,
		&rep.Influence,
		&rep.Priority,
		&rep.CreatedAt,
		&rep.UpdatedAt,
	)
	if err != nil {
		return ports.ReportRecord{}, err
	}
	return rep, nil
}
