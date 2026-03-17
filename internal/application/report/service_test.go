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
	if _, ok := m.byID[r.ID]; ok {
		return errors.New("duplicate id")
	}
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

type fakeClock2 struct{ t time.Time }

func (c fakeClock2) Now() time.Time { return c.t }

type fakeRandom2 struct{ n int }

func (r *fakeRandom2) NewID() string {
	r.n++
	return "r-" + string(rune('0'+r.n))
}

func (r *fakeRandom2) NewToken() (string, error) { return "unused", nil }

func TestService_CreateAndGetUserReport(t *testing.T) {
	repo := &memReports{byID: map[string]ports.ReportRecord{}}
	clk := fakeClock2{t: time.Unix(1_700_000_000, 0).UTC()}
	rnd := &fakeRandom2{}

	s := NewService(Deps{
		Reports: repo,
		Clock:   clk,
		Random:  rnd,
	})

	ctx := context.Background()
	created, err := s.Create(ctx, CreateRequest{
		UserID:      "u1",
		Title:       "Crash on login",
		Description: "Steps...",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.Status != StatusNew {
		t.Fatalf("expected status new, got %q", created.Status)
	}

	got, err := s.GetForUser(ctx, "u1", created.ID)
	if err != nil {
		t.Fatalf("GetForUser error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected same id")
	}

	_, err = s.GetForUser(ctx, "u2", created.ID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestService_ModeratorCanChangeStatus(t *testing.T) {
	repo := &memReports{byID: map[string]ports.ReportRecord{}}
	now := time.Unix(1_700_000_000, 0).UTC()
	clk := fakeClock2{t: now}
	rnd := &fakeRandom2{}
	_ = repo.Create(context.Background(), ports.ReportRecord{
		ID:          "r-1",
		UserID:      "u1",
		Title:       "t",
		Description: "d",
		Status:      StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	s := NewService(Deps{Reports: repo, Clock: clk, Random: rnd})

	err := s.ChangeStatus(context.Background(), ChangeStatusRequest{
		ActorRole: "moderator",
		ReportID:  "r-1",
		Status:    StatusResolved,
	})
	if err != nil {
		t.Fatalf("ChangeStatus error: %v", err)
	}

	r, _, _ := repo.GetByID(context.Background(), "r-1")
	if r.Status != StatusResolved {
		t.Fatalf("expected status resolved, got %q", r.Status)
	}
}
