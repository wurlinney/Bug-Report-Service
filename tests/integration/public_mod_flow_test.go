//go:build integration

package integration

import (
	"context"
	"testing"

	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/adapters/security"
	"bug-report-service/internal/application/note"
	"bug-report-service/internal/application/ports"
	"bug-report-service/internal/application/report"
)

func TestIntegration_PublicCreate_AndModeratorNotes(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	// create moderator directly (manual provisioning style)
	hasher := security.NewBCryptPasswordHasher(4)
	pw, err := hasher.HashPassword("pass-12345")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	users := postgres.NewModeratorRepository(db)
	createdModerator, err := users.Create(context.Background(), ports.UserRecord{
		Name:         "Alice Moderator",
		Email:        "mod@example.com",
		PasswordHash: pw,
	})
	if err != nil {
		t.Fatalf("create moderator: %v", err)
	}

	reportsRepo := postgres.NewReportRepository(db)
	reportSvc := report.NewService(report.Deps{
		Reports: reportsRepo,
	})

	rep, err := reportSvc.Create(context.Background(), report.CreateRequest{
		ReporterName: "Ivan Ivanov",
		Description:  "Something is broken",
	})
	if err != nil {
		t.Fatalf("create public report: %v", err)
	}

	noteRepo := postgres.NewNoteRepository(db)
	noteSvc := note.NewService(note.Deps{
		Notes:   noteRepo,
		Reports: reportsRepo,
	})

	createdNote, err := noteSvc.Create(context.Background(), note.CreateRequest{
		ActorRole: "moderator",
		ActorID:   createdModerator.ID,
		ReportID:  rep.ID,
		Text:      "Investigating",
	})
	if err != nil {
		t.Fatalf("create internal note: %v", err)
	}
	if createdNote.ReportID != rep.ID {
		t.Fatalf("note/report mismatch: %+v", createdNote)
	}

	items, total, err := noteSvc.List(context.Background(), note.ListRequest{
		ActorRole: "moderator",
		ReportID:  rep.ID,
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Text != "Investigating" {
		t.Fatalf("unexpected notes: total=%d items=%+v", total, items)
	}
}
