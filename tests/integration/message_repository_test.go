//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/application/ports"
)

func TestPostgresMessageRepository_CreateAndList(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	users := postgres.NewUserRepository(db)
	reports := postgres.NewReportRepository(db)
	msgs := postgres.NewMessageRepository(db)

	now := time.Unix(1_700_000_000, 0).UTC()
	_ = users.Create(context.Background(), ports.UserRecord{
		ID:           "u1",
		Email:        "a@example.com",
		PasswordHash: "x",
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	_ = reports.Create(context.Background(), ports.ReportRecord{
		ID:          "r1",
		UserID:      "u1",
		Title:       "t",
		Description: "d",
		Status:      "new",
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	_ = msgs.Create(context.Background(), ports.MessageRecord{
		ID:         "m1",
		ReportID:   "r1",
		SenderID:   "u1",
		SenderRole: "user",
		Text:       "hello",
		CreatedAt:  now.Add(2 * time.Second),
	})
	_ = msgs.Create(context.Background(), ports.MessageRecord{
		ID:         "m2",
		ReportID:   "r1",
		SenderID:   "u1",
		SenderRole: "user",
		Text:       "world",
		CreatedAt:  now.Add(1 * time.Second),
	})

	items, total, err := msgs.ListByReport(context.Background(), "r1", ports.MessageListFilter{
		Limit:    10,
		Offset:   0,
		SortDesc: false,
	})
	if err != nil {
		t.Fatalf("ListByReport: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("unexpected total/len: %d/%d", total, len(items))
	}
	if items[0].ID != "m2" || items[1].ID != "m1" {
		t.Fatalf("unexpected order: %s,%s", items[0].ID, items[1].ID)
	}
}
