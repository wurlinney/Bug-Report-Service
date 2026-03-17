package message

import (
	"context"
	"time"

	"bug-report-service/internal/application/ports"
)

// Local minimal report repo for message tests (kept in _test to avoid coupling packages).
type memReports struct {
	byID map[string]ports.ReportRecord
}

func (m *memReports) Create(_ context.Context, r ports.ReportRecord) error {
	m.byID[r.ID] = r
	return nil
}

func (m *memReports) GetByID(_ context.Context, id string) (ports.ReportRecord, bool, error) {
	r, ok := m.byID[id]
	return r, ok, nil
}

func (m *memReports) UpdateStatus(_ context.Context, id string, status string, updatedAt time.Time) error {
	r, ok := m.byID[id]
	if !ok {
		return ports.ErrNotFound
	}
	r.Status = status
	r.UpdatedAt = updatedAt
	m.byID[id] = r
	return nil
}

func (m *memReports) ListByUser(_ context.Context, userID string, f ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	var out []ports.ReportRecord
	for _, r := range m.byID {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return ports.ApplyReportListFilter(out, f)
}

func (m *memReports) ListAll(_ context.Context, f ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	var out []ports.ReportRecord
	for _, r := range m.byID {
		out = append(out, r)
	}
	return ports.ApplyReportListFilter(out, f)
}
