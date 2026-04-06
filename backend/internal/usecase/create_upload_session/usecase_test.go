package create_upload_session

import (
	"context"
	"testing"
	"time"

	"bug-report-service/internal/domain/uploadsession"
)

type mockSessionCreator struct {
	session uploadsession.UploadSession
	err     error
}

func (m *mockSessionCreator) Create(_ context.Context) (uploadsession.UploadSession, error) {
	return m.session, m.err
}

func TestExecute_Success(t *testing.T) {
	now := time.Now()
	uc := New(&mockSessionCreator{
		session: uploadsession.UploadSession{ID: "sess-1", CreatedAt: now},
	})

	got, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "sess-1" {
		t.Errorf("expected ID sess-1, got %s", got.ID)
	}
}
