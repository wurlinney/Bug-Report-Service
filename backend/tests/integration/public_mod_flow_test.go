//go:build integration

package integration

import (
	"context"
	"testing"

	"bug-report-service/internal/domain/user"
	"bug-report-service/internal/infrastructure/auth"
	dbattachment "bug-report-service/internal/infrastructure/database/attachment"
	dbnote "bug-report-service/internal/infrastructure/database/note"
	dbreport "bug-report-service/internal/infrastructure/database/report"
	dbuploadsession "bug-report-service/internal/infrastructure/database/uploadsession"
	dbuser "bug-report-service/internal/infrastructure/database/user"
	createnote "bug-report-service/internal/usecase/create_note"
	createreport "bug-report-service/internal/usecase/create_report"
	listnotes "bug-report-service/internal/usecase/list_notes"
)

func TestIntegration_PublicCreate_AndModeratorNotes(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	hasher := auth.NewBCryptPasswordHasher(4)
	pw, err := hasher.HashPassword("pass-12345")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	users := dbuser.NewRepository(db)
	createdModerator, err := users.Create(context.Background(), user.User{
		Name:         "Alice Moderator",
		Email:        "mod@example.com",
		PasswordHash: pw,
	})
	if err != nil {
		t.Fatalf("create moderator: %v", err)
	}

	reportsRepo := dbreport.NewRepository(db)
	uploadSessionsRepo := dbuploadsession.NewRepository(db)
	attsRepo := dbattachment.NewRepository(db)

	type sessionChecker struct {
		repo *dbuploadsession.Repository
	}
	checker := &sessionCheckerAdapter{repo: uploadSessionsRepo}

	createReportUC := createreport.New(reportsRepo, checker, attsRepo)
	rep, err := createReportUC.Execute(context.Background(), createreport.Request{
		ReporterName: "Ivan Ivanov",
		Description:  "Something is broken",
	})
	if err != nil {
		t.Fatalf("create public report: %v", err)
	}

	noteRepo := dbnote.NewRepository(db)
	createNoteUC := createnote.New(noteRepo, reportsRepo)

	created, err := createNoteUC.Execute(context.Background(), createnote.Request{
		ActorRole: "moderator",
		ActorID:   createdModerator.ID,
		ReportID:  rep.ID,
		Text:      "Investigating",
	})
	if err != nil {
		t.Fatalf("create internal note: %v", err)
	}
	if created.ReportID != rep.ID {
		t.Fatalf("note/report mismatch: %+v", created)
	}

	listNotesUC := listnotes.New(noteRepo, reportsRepo)
	items, total, err := listNotesUC.Execute(context.Background(), listnotes.Request{
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

type sessionCheckerAdapter struct {
	repo *dbuploadsession.Repository
}

func (a *sessionCheckerAdapter) GetByID(ctx context.Context, id string) (bool, error) {
	_, found, err := a.repo.GetByID(ctx, id)
	return found, err
}
