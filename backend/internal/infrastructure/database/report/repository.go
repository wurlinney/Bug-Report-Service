package report

import (
	"context"
	"fmt"
	"strings"

	"bug-report-service/internal/domain"
	domainreport "bug-report-service/internal/domain/report"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

type scanner interface {
	Scan(dest ...any) error
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, rep domainreport.Report) (domainreport.Report, error) {
	const q = `
INSERT INTO bug_reports (reporter_name, description, status)
VALUES ($1,$2,$3)
RETURNING id::text, reporter_name, description, status, influence, priority, created_at, updated_at
`
	created, err := scanReport(r.db.QueryRow(ctx, q, rep.ReporterName, rep.Description, rep.Status))
	if err != nil {
		return domainreport.Report{}, err
	}
	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domainreport.Report, bool, error) {
	const q = `
SELECT r.id::text, r.reporter_name, r.description, r.status, r.influence, r.priority, r.created_at, r.updated_at
FROM bug_reports r
WHERE r.id = $1::bigint
`
	rep, err := scanReport(r.db.QueryRow(ctx, q, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domainreport.Report{}, false, nil
		}
		return domainreport.Report{}, false, err
	}
	return rep, true, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status string) error {
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
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) UpdateMeta(ctx context.Context, id string, priority string, influence string) error {
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
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListAll(ctx context.Context, f domainreport.ListFilter) ([]domainreport.Report, int, error) {
	return r.list(ctx, f)
}

func (r *Repository) list(ctx context.Context, f domainreport.ListFilter) ([]domainreport.Report, int, error) {
	where, args := buildWhere(f)

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

	totalQ := "SELECT COUNT(*) FROM bug_reports r" + where
	var total int
	if err := r.db.QueryRow(ctx, totalQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

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

	var out []domainreport.Report
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

func buildWhere(f domainreport.ListFilter) (string, []any) {
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

func scanReport(s scanner) (domainreport.Report, error) {
	var rep domainreport.Report
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
		return domainreport.Report{}, err
	}
	return rep, nil
}
