package ports

import (
	"context"
	"sort"
	"time"
)

type MessageRepository interface {
	Create(ctx context.Context, msg MessageRecord) error
	ListByReport(ctx context.Context, reportID string, f MessageListFilter) (items []MessageRecord, total int, err error)
}

type MessageRecord struct {
	ID         string
	ReportID   string
	SenderID   string
	SenderRole string
	Text       string
	CreatedAt  time.Time
}

type MessageListFilter struct {
	Limit    int
	Offset   int
	SortDesc bool
}

func ApplyMessageListFilter(items []MessageRecord, f MessageListFilter) ([]MessageRecord, int, error) {
	total := len(items)

	sort.Slice(items, func(i, j int) bool {
		less := items[i].CreatedAt.Before(items[j].CreatedAt)
		if f.SortDesc {
			return !less
		}
		return less
	})

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []MessageRecord{}, total, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], total, nil
}
