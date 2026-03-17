package ports

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type ReportRepository interface {
	Create(ctx context.Context, r ReportRecord) error
	GetByID(ctx context.Context, id string) (r ReportRecord, found bool, err error)
	UpdateStatus(ctx context.Context, id string, status string, updatedAt time.Time) error
	ListByUser(ctx context.Context, userID string, f ReportListFilter) (items []ReportRecord, total int, err error)
	ListAll(ctx context.Context, f ReportListFilter) (items []ReportRecord, total int, err error)
}

type ReportRecord struct {
	ID          string
	UserID      string
	UserName    string
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ReportListFilter struct {
	Status    *string
	UserID    *string
	Query     *string
	CreatedAt *TimeRange

	SortBy   string // created_at|updated_at
	SortDesc bool
	Limit    int
	Offset   int
}

type TimeRange struct {
	From *time.Time
	To   *time.Time
}

// ApplyReportListFilter is a helper for in-memory repos/tests.
// The real implementation for Postgres will do this in SQL.
func ApplyReportListFilter(items []ReportRecord, f ReportListFilter) ([]ReportRecord, int, error) {
	var filtered []ReportRecord
	for _, r := range items {
		if f.Status != nil && r.Status != *f.Status {
			continue
		}
		if f.UserID != nil && r.UserID != *f.UserID {
			continue
		}
		if f.CreatedAt != nil {
			if f.CreatedAt.From != nil && r.CreatedAt.Before(*f.CreatedAt.From) {
				continue
			}
			if f.CreatedAt.To != nil && r.CreatedAt.After(*f.CreatedAt.To) {
				continue
			}
		}
		if f.Query != nil {
			q := strings.ToLower(strings.TrimSpace(*f.Query))
			if q != "" {
				hay := strings.ToLower(r.Title + " " + r.Description)
				if !strings.Contains(hay, q) {
					continue
				}
			}
		}
		filtered = append(filtered, r)
	}

	total := len(filtered)

	sort.Slice(filtered, func(i, j int) bool {
		var less bool
		switch f.SortBy {
		case "updated_at":
			less = filtered[i].UpdatedAt.Before(filtered[j].UpdatedAt)
		default:
			less = filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
		}
		if f.SortDesc {
			return !less
		}
		return less
	})

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(filtered) {
		return []ReportRecord{}, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}
