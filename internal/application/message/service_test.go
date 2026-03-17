package message

import (
	"context"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/application/ports"
)

type memMessages struct {
	byReportID map[string][]ports.MessageRecord
}

func (m *memMessages) Create(_ context.Context, msg ports.MessageRecord) error {
	m.byReportID[msg.ReportID] = append(m.byReportID[msg.ReportID], msg)
	return nil
}

func (m *memMessages) ListByReport(_ context.Context, reportID string, f ports.MessageListFilter) ([]ports.MessageRecord, int, error) {
	items := append([]ports.MessageRecord(nil), m.byReportID[reportID]...)
	return ports.ApplyMessageListFilter(items, f)
}

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

type fakeRandom struct{ n int }

func (r *fakeRandom) NewID() string {
	r.n++
	return "m" + string(rune('0'+r.n))
}
func (r *fakeRandom) NewToken() (string, error) { return "unused", nil }

func TestService_UserCanPostAndListOnlyOwnReport(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reports := &memReports{byID: map[string]ports.ReportRecord{
		"r1": {
			ID:          "r1",
			UserID:      "u1",
			Title:       "t",
			Description: "d",
			Status:      "new",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}}
	msgs := &memMessages{byReportID: map[string][]ports.MessageRecord{}}

	s := NewService(Deps{
		Reports:  reports,
		Messages: msgs,
		Clock:    fakeClock{t: now},
		Random:   &fakeRandom{},
	})

	created, err := s.Create(context.Background(), CreateRequest{
		ActorRole: "user",
		ActorID:   "u1",
		ReportID:  "r1",
		Text:      "hello",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.SenderRole != "user" || created.SenderID != "u1" {
		t.Fatalf("unexpected sender fields")
	}

	_, err = s.List(context.Background(), ListRequest{
		ActorRole: "user",
		ActorID:   "u2",
		ReportID:  "r1",
		Limit:     20,
		Offset:    0,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestService_ModeratorCanPostAndListAnyReport(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reports := &memReports{byID: map[string]ports.ReportRecord{
		"r1": {ID: "r1", UserID: "u1", Title: "t", Description: "d", Status: "new", CreatedAt: now, UpdatedAt: now},
	}}
	msgs := &memMessages{byReportID: map[string][]ports.MessageRecord{}}

	s := NewService(Deps{
		Reports:  reports,
		Messages: msgs,
		Clock:    fakeClock{t: now},
		Random:   &fakeRandom{},
	})

	_, err := s.Create(context.Background(), CreateRequest{
		ActorRole: "moderator",
		ActorID:   "m1",
		ReportID:  "r1",
		Text:      "please provide steps",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	list, err := s.List(context.Background(), ListRequest{
		ActorRole: "moderator",
		ActorID:   "m1",
		ReportID:  "r1",
		Limit:     20,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(list.Items))
	}
	if list.Total != 1 {
		t.Fatalf("expected total=1, got %d", list.Total)
	}
}

func TestService_ListPaginationSort(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	reports := &memReports{byID: map[string]ports.ReportRecord{
		"r1": {ID: "r1", UserID: "u1", Title: "t", Description: "d", Status: "new", CreatedAt: base, UpdatedAt: base},
	}}
	msgs := &memMessages{byReportID: map[string][]ports.MessageRecord{
		"r1": {
			{ID: "m1", ReportID: "r1", SenderID: "u1", SenderRole: "user", Text: "1", CreatedAt: base.Add(2 * time.Second)},
			{ID: "m2", ReportID: "r1", SenderID: "u1", SenderRole: "user", Text: "2", CreatedAt: base.Add(1 * time.Second)},
			{ID: "m3", ReportID: "r1", SenderID: "u1", SenderRole: "user", Text: "3", CreatedAt: base.Add(3 * time.Second)},
		},
	}}

	s := NewService(Deps{
		Reports:  reports,
		Messages: msgs,
		Clock:    fakeClock{t: base},
		Random:   &fakeRandom{},
	})

	resp, err := s.List(context.Background(), ListRequest{
		ActorRole: "user",
		ActorID:   "u1",
		ReportID:  "r1",
		Limit:     2,
		Offset:    0,
		SortDesc:  false,
	})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].ID != "m2" || resp.Items[1].ID != "m1" {
		t.Fatalf("unexpected order: %v,%v", resp.Items[0].ID, resp.Items[1].ID)
	}
}
