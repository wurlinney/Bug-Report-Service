//go:build integration

package integration

import (
	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/application/ports"
	"context"
	"testing"
	"time"
)

func TestPostgresAttachmentRepository_CreateGetIdempotencyAndList(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	users := postgres.NewUserRepository(db)
	reports := postgres.NewReportRepository(db)
	atts := postgres.NewAttachmentRepository(db)

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

	a1 := ports.AttachmentRecord{
		ID:             "a1",
		ReportID:       "r1",
		FileName:       "x.png",
		ContentType:    "image/png",
		FileSize:       10,
		StorageKey:     "k1",
		CreatedAt:      now,
		IdempotencyKey: "idem-1",
		UploadedByID:   "u1",
		UploadedByRole: "user",
	}
	if err := atts.Create(context.Background(), a1); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, found, err := atts.GetByIdempotencyKey(context.Background(), "r1", "idem-1")
	if err != nil {
		t.Fatalf("GetByIdempotencyKey: %v", err)
	}
	if !found || got.ID != "a1" || got.StorageKey != "k1" {
		t.Fatalf("unexpected: found=%v got=%+v", found, got)
	}

	list, err := atts.ListByReport(context.Background(), "r1")
	if err != nil {
		t.Fatalf("ListByReport: %v", err)
	}
	if len(list) != 1 || list[0].ID != "a1" {
		t.Fatalf("unexpected list: %+v", list)
	}
}
