//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/adapters/httpserver"
	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/application/attachment"
	"bug-report-service/internal/application/message"
	"bug-report-service/internal/application/ports"
	"bug-report-service/internal/application/report"
)

type fakeVerifier struct{}

func (v fakeVerifier) VerifyAccessToken(token string) (httpserver.Principal, error) {
	switch token {
	case "u-token":
		return httpserver.Principal{UserID: "u1", Role: "user"}, nil
	case "m-token":
		return httpserver.Principal{UserID: "m1", Role: "moderator"}, nil
	default:
		return httpserver.Principal{}, httpserver.ErrUnauthorized
	}
}

type fakeSigner struct{}

func (s fakeSigner) PresignGetObject(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://example.test/" + key, nil
}

type fixedClockHTTP struct{ t time.Time }

func (c fixedClockHTTP) Now() time.Time { return c.t }

type seqRandomHTTP struct {
	ids []string
	i   int
}

func (r *seqRandomHTTP) NewID() string {
	if r.i >= len(r.ids) {
		return "id-x"
	}
	id := r.ids[r.i]
	r.i++
	return id
}

func (r *seqRandomHTTP) NewToken() (string, error) { return "unused", nil }

func TestIntegration_HTTPFlow_UserAndModerator(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	now := time.Unix(1_700_000_000, 0).UTC()
	ctx := context.Background()

	usersRepo := postgres.NewUserRepository(db)
	reportsRepo := postgres.NewReportRepository(db)
	attsRepo := postgres.NewAttachmentRepository(db)
	msgRepo := postgres.NewMessageRepository(db)

	// Seed user + moderator (FK on bug_reports.user_id).
	if err := usersRepo.Create(ctx, ports.UserRecord{ID: "u1", Email: "u1@example.com", PasswordHash: "x", Role: "user", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := usersRepo.Create(ctx, ports.UserRecord{ID: "m1", Email: "m1@example.com", PasswordHash: "x", Role: "moderator", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("seed moderator: %v", err)
	}

	reportSvc := report.NewService(report.Deps{
		Reports: reportsRepo,
		Clock:   fixedClockHTTP{t: now},
		Random:  &seqRandomHTTP{ids: []string{"r-http-1"}},
	})
	msgSvc := message.NewService(message.Deps{
		Reports:  reportsRepo,
		Messages: msgRepo,
		Clock:    fixedClockHTTP{t: now},
		Random:   &seqRandomHTTP{ids: []string{"m-http-1"}},
	})
	attSvc := attachment.NewService(attachment.Deps{
		Reports:      reportsRepo,
		Attachments:  attsRepo,
		Storage:      nil,
		Clock:        fixedClockHTTP{t: now},
		Random:       &seqRandomHTTP{ids: []string{"unused"}},
		MaxFileSize:  20 * 1024 * 1024,
		AllowedMIMEs: map[string]struct{}{"image/png": {}, "image/jpeg": {}, "image/webp": {}},
	})

	api := httpserver.NewAPI(httpserver.Deps{
		Ready:             httpserver.NewReadiness(),
		ReportService:     reportSvc,
		MessageService:    msgSvc,
		AttachmentService: attSvc,
		AttachmentSigner:  fakeSigner{},
		TokenVerifier:     fakeVerifier{},
	})
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	// 1) user creates report
	{
		body, _ := json.Marshal(map[string]any{"title": "Crash", "description": "Steps..."})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer u-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /reports: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		var created struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&created)
		if created.ID != "r-http-1" {
			t.Fatalf("unexpected report id: %+v", created)
		}
	}

	// 2) moderator lists reports
	{
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/mod/reports?sort_desc=true", nil)
		req.Header.Set("Authorization", "Bearer m-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /mod/reports: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var out struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
			Total int `json:"total"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		if out.Total != 1 || len(out.Items) != 1 || out.Items[0].ID != "r-http-1" {
			t.Fatalf("unexpected list: %+v", out)
		}
	}

	// 3) moderator changes status
	{
		body, _ := json.Marshal(map[string]any{"status": "in_review"})
		req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/mod/reports/r-http-1/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer m-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("PATCH /mod/reports/{id}/status: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	}

	// 4) moderator creates message
	{
		body, _ := json.Marshal(map[string]any{"text": "hello from mod"})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/mod/reports/r-http-1/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer m-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /mod/reports/{id}/messages: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
	}

	// 5) finalize attachment (simulate tusd completion) and list attachments
	{
		if _, err := attSvc.Finalize(ctx, attachment.FinalizeRequest{
			ActorRole:      "moderator",
			ActorID:        "m1",
			ReportID:       "r-http-1",
			UploadID:       "upl-http-1",
			FileName:       "x.png",
			ContentType:    "image/png",
			FileSize:       123,
			StorageKey:     "tus/upl-http-1",
			IdempotencyKey: "idem-http-1",
		}); err != nil {
			t.Fatalf("Finalize: %v", err)
		}

		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/mod/reports/r-http-1/attachments", nil)
		req.Header.Set("Authorization", "Bearer m-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /mod/reports/{id}/attachments: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var out struct {
			Items []struct {
				ID          string `json:"id"`
				DownloadURL string `json:"download_url"`
			} `json:"items"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		if len(out.Items) != 1 || out.Items[0].ID != "upl-http-1" || out.Items[0].DownloadURL == "" {
			t.Fatalf("unexpected attachments: %+v", out)
		}
	}
}
