package report

import (
	"context"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/application/ports"
)

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

func (m *memReports) UpdateStatus(_ context.Context, id string, status string, when time.Time) error {
	r, ok := m.byID[id]
	if !ok {
		return ports.ErrNotFound
	}
	r.Status = status
	r.UpdatedAt = when
	m.byID[id] = r
	return nil
}

func (m *memReports) ListAll(_ context.Context, f ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	var out []ports.ReportRecord
	for _, r := range m.byID {
		out = append(out, r)
	}
	return ports.ApplyReportListFilter(out, f)
}

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

type fakeRandom struct{}

func (r fakeRandom) NewID() string             { return "r1" }
func (r fakeRandom) NewToken() (string, error) { return "unused", nil }

func TestService_CreatePublicReport(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	repo := &memReports{byID: map[string]ports.ReportRecord{}}
	s := NewService(Deps{
		Reports: repo,
		Clock:   fakeClock{t: now},
		Random:  fakeRandom{},
	})

	got, err := s.Create(context.Background(), CreateRequest{
		ReporterName: "Ivan Ivanov",
		Description:  "UI freezes",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if got.ID != "r1" || got.ReporterName != "Ivan Ivanov" || got.Status != StatusNew {
		t.Fatalf("unexpected dto: %+v", got)
	}
}

func TestService_ListAll_ForModerator(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	repo := &memReports{byID: map[string]ports.ReportRecord{
		"r1": {ID: "r1", ReporterName: "Ivan", Description: "bug", Status: StatusNew, CreatedAt: now, UpdatedAt: now},
	}}
	s := NewService(Deps{
		Reports: repo,
		Clock:   fakeClock{t: now},
		Random:  fakeRandom{},
	})

	items, total, err := s.ListAll(context.Background(), ListAllRequest{
		ActorRole: "moderator",
		Limit:     20,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("ListAll error: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("unexpected result: total=%d len=%d", total, len(items))
	}
}

func TestService_GetForActor_ForbiddenForNonModerator(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	repo := &memReports{byID: map[string]ports.ReportRecord{
		"r1": {ID: "r1", ReporterName: "Ivan", Description: "bug", Status: StatusNew, CreatedAt: now, UpdatedAt: now},
	}}
	s := NewService(Deps{
		Reports: repo,
		Clock:   fakeClock{t: now},
		Random:  fakeRandom{},
	})

	_, err := s.GetForActor(context.Background(), "user", "u1", "r1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
