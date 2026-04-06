package report

import (
	"sort"
	"strings"
	"time"
)

type Report struct {
	ID           string
	ReporterName string
	Description  string
	Status       string
	Influence    string
	Priority     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ListFilter struct {
	Status       *string
	ReporterName *string
	Query        *string
	CreatedAt    *TimeRange

	SortBy   string // created_at|updated_at
	SortDesc bool
	Limit    int
	Offset   int
}

type TimeRange struct {
	From *time.Time
	To   *time.Time
}

// ApplyListFilter is a helper for in-memory repos/tests.
func ApplyListFilter(items []Report, f ListFilter) ([]Report, int, error) {
	var filtered []Report
	for _, r := range items {
		if f.Status != nil && r.Status != *f.Status {
			continue
		}
		if f.ReporterName != nil && r.ReporterName != *f.ReporterName {
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
				hay := strings.ToLower(r.ReporterName + " " + r.Description)
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
		return []Report{}, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}
