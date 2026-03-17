//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/application/ports"
)

func TestPostgresReportRepository_CreateGetUpdateStatusAndList(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	users := postgres.NewUserRepository(db)
	reports := postgres.NewReportRepository(db)

	now := time.Unix(1_700_000_000, 0).UTC()
	if err := users.Create(context.Background(), ports.UserRecord{
		ID:           "u1",
		Email:        "a@example.com",
		PasswordHash: "x",
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	r1 := ports.ReportRecord{
		ID:          "r1",
		UserID:      "u1",
		Title:       "Crash on login",
		Description: "Steps",
		Status:      "new",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := reports.Create(context.Background(), r1); err != nil {
		t.Fatalf("Create report: %v", err)
	}

	got, found, err := reports.GetByID(context.Background(), "r1")
	if err != nil || !found {
		t.Fatalf("GetByID err=%v found=%v", err, found)
	}
	if got.Title != r1.Title || got.UserID != "u1" {
		t.Fatalf("unexpected report: %+v", got)
	}

	updAt := now.Add(10 * time.Minute)
	if err := reports.UpdateStatus(context.Background(), "r1", "resolved", updAt); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, found, _ = reports.GetByID(context.Background(), "r1")
	if !found || got.Status != "resolved" || !got.UpdatedAt.Equal(updAt) {
		t.Fatalf("unexpected updated report: %+v", got)
	}

	// Add another report to test list.
	r2 := r1
	r2.ID = "r2"
	r2.Title = "UI glitch"
	r2.Status = "new"
	r2.CreatedAt = now.Add(1 * time.Hour)
	r2.UpdatedAt = r2.CreatedAt
	if err := reports.Create(context.Background(), r2); err != nil {
		t.Fatalf("Create report2: %v", err)
	}

	status := "new"
	items, total, err := reports.ListByUser(context.Background(), "u1", ports.ReportListFilter{
		Status:   &status,
		SortBy:   "created_at",
		SortDesc: true,
		Limit:    10,
		Offset:   0,
	})
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != "r2" {
		t.Fatalf("unexpected list result total=%d items=%v", total, ids(items))
	}
}

func ids(rs []ports.ReportRecord) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.ID)
	}
	return out
}
