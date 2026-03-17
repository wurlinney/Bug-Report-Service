//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/application/attachment"
	"bug-report-service/internal/application/message"
	"bug-report-service/internal/application/ports"
	"bug-report-service/internal/application/report"
)

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

type seqRandom struct {
	ids []string
	i   int
}

func (r *seqRandom) NewID() string {
	if r.i >= len(r.ids) {
		return "id-x"
	}
	id := r.ids[r.i]
	r.i++
	return id
}

func (r *seqRandom) NewToken() (string, error) { return "unused", nil }

func TestIntegration_Service_ModeratorListAndChangeStatus(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	usersRepo := postgres.NewUserRepository(db)
	reportsRepo := postgres.NewReportRepository(db)

	now := time.Unix(1_700_000_000, 0).UTC()
	ctx := context.Background()

	_ = usersRepo.Create(ctx, ports.UserRecord{ID: "u1", Email: "u1@example.com", PasswordHash: "x", Role: "user", CreatedAt: now, UpdatedAt: now})
	_ = usersRepo.Create(ctx, ports.UserRecord{ID: "u2", Email: "u2@example.com", PasswordHash: "x", Role: "user", CreatedAt: now, UpdatedAt: now})

	_ = reportsRepo.Create(ctx, ports.ReportRecord{ID: "r1", UserID: "u1", Title: "Crash on login", Description: "Steps", Status: report.StatusNew, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)})
	_ = reportsRepo.Create(ctx, ports.ReportRecord{ID: "r2", UserID: "u2", Title: "UI glitch", Description: "Details", Status: report.StatusResolved, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)})

	svc := report.NewService(report.Deps{
		Reports: reportsRepo,
		Clock:   fakeClock{t: now},
		Random:  &seqRandom{ids: []string{"unused"}},
	})

	// Moderator can list all.
	items, total, err := svc.ListAll(ctx, report.ListAllRequest{
		ActorRole: "moderator",
		SortBy:    "created_at",
		SortDesc:  true,
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if total != 2 || len(items) != 2 || items[0].ID != "r2" || items[1].ID != "r1" {
		t.Fatalf("unexpected list: total=%d ids=%v", total, []string{items[0].ID, items[1].ID})
	}

	// User cannot list all.
	if _, _, err := svc.ListAll(ctx, report.ListAllRequest{ActorRole: "user"}); err != report.ErrForbidden {
		t.Fatalf("expected ErrForbidden for user, got %v", err)
	}

	// Moderator can change status.
	if err := svc.ChangeStatus(ctx, report.ChangeStatusRequest{
		ActorRole: "moderator",
		ReportID:  "r1",
		Status:    report.StatusInReview,
	}); err != nil {
		t.Fatalf("ChangeStatus: %v", err)
	}
	got, found, err := reportsRepo.GetByID(ctx, "r1")
	if err != nil || !found {
		t.Fatalf("GetByID: err=%v found=%v", err, found)
	}
	if got.Status != report.StatusInReview {
		t.Fatalf("status not updated: %+v", got)
	}
}

func TestIntegration_Service_ModeratorMessagesAndAttachmentFinalize(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	usersRepo := postgres.NewUserRepository(db)
	reportsRepo := postgres.NewReportRepository(db)
	msgRepo := postgres.NewMessageRepository(db)
	attRepo := postgres.NewAttachmentRepository(db)

	now := time.Unix(1_700_000_000, 0).UTC()
	ctx := context.Background()

	_ = usersRepo.Create(ctx, ports.UserRecord{ID: "u1", Email: "u1@example.com", PasswordHash: "x", Role: "user", CreatedAt: now, UpdatedAt: now})
	_ = usersRepo.Create(ctx, ports.UserRecord{ID: "m1", Email: "m1@example.com", PasswordHash: "x", Role: "moderator", CreatedAt: now, UpdatedAt: now})

	_ = reportsRepo.Create(ctx, ports.ReportRecord{ID: "r1", UserID: "u1", Title: "t", Description: "d", Status: report.StatusNew, CreatedAt: now, UpdatedAt: now})

	msgSvc := message.NewService(message.Deps{
		Reports:  reportsRepo,
		Messages: msgRepo,
		Clock:    fakeClock{t: now},
		Random:   &seqRandom{ids: []string{"msg-1"}},
	})

	// Moderator can post message to any report.
	created, err := msgSvc.Create(ctx, message.CreateRequest{
		ActorRole: "moderator",
		ActorID:   "m1",
		ReportID:  "r1",
		Text:      "hello from mod",
	})
	if err != nil {
		t.Fatalf("Create message: %v", err)
	}
	if created.ID != "msg-1" || created.SenderRole != "moderator" {
		t.Fatalf("unexpected created message: %+v", created)
	}

	list, err := msgSvc.List(ctx, message.ListRequest{
		ActorRole: "moderator",
		ActorID:   "m1",
		ReportID:  "r1",
		Limit:     10,
		Offset:    0,
		SortDesc:  false,
	})
	if err != nil {
		t.Fatalf("List messages: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].Text != "hello from mod" {
		t.Fatalf("unexpected list: %+v", list)
	}

	attSvc := attachment.NewService(attachment.Deps{
		Reports:      reportsRepo,
		Attachments:  attRepo,
		Storage:      nil, // Finalize does not use storage
		Clock:        fakeClock{t: now},
		Random:       &seqRandom{ids: []string{"unused"}},
		MaxFileSize:  20 * 1024 * 1024,
		AllowedMIMEs: map[string]struct{}{"image/png": {}, "image/jpeg": {}, "image/webp": {}},
	})

	// Finalize is idempotent via idempotency_key (and upload_id as attachment id).
	a1, err := attSvc.Finalize(ctx, attachment.FinalizeRequest{
		ActorRole:      "moderator",
		ActorID:        "m1",
		ReportID:       "r1",
		UploadID:       "upl-1",
		FileName:       "x.png",
		ContentType:    "image/png",
		FileSize:       123,
		StorageKey:     "tus/upl-1",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	a2, err := attSvc.Finalize(ctx, attachment.FinalizeRequest{
		ActorRole:      "moderator",
		ActorID:        "m1",
		ReportID:       "r1",
		UploadID:       "upl-1",
		FileName:       "x.png",
		ContentType:    "image/png",
		FileSize:       123,
		StorageKey:     "tus/upl-1",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("Finalize retry: %v", err)
	}
	if a1.ID != a2.ID || a2.ID != "upl-1" {
		t.Fatalf("expected idempotent finalize, got a1=%+v a2=%+v", a1, a2)
	}

	items, err := attSvc.ListForReport(ctx, attachment.ListForReportRequest{
		ActorRole: "moderator",
		ActorID:   "m1",
		ReportID:  "r1",
	})
	if err != nil {
		t.Fatalf("ListForReport: %v", err)
	}
	if len(items) != 1 || items[0].ID != "upl-1" {
		t.Fatalf("unexpected attachments: %+v", items)
	}
}
