package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/application/attachment"
	"bug-report-service/internal/application/auth"
	"bug-report-service/internal/application/message"
	"bug-report-service/internal/application/ports"
	"bug-report-service/internal/application/report"
	appuser "bug-report-service/internal/application/user"
)

type memUsers struct {
	byEmail map[string]ports.UserRecord
	byID    map[string]ports.UserRecord
}

func (m *memUsers) GetByEmail(_ context.Context, email string) (ports.UserRecord, bool, error) {
	u, ok := m.byEmail[email]
	return u, ok, nil
}

func (m *memUsers) GetByID(_ context.Context, id string) (ports.UserRecord, bool, error) {
	u, ok := m.byID[id]
	return u, ok, nil
}

func (m *memUsers) Create(_ context.Context, u ports.UserRecord) error {
	if _, exists := m.byEmail[u.Email]; exists {
		return ports.ErrUniqueViolation
	}
	m.byEmail[u.Email] = u
	m.byID[u.ID] = u
	return nil
}

type memRefresh struct {
	byTokenID map[string]ports.RefreshTokenRecord
}

func (m *memRefresh) Save(_ context.Context, rt ports.RefreshTokenRecord) error {
	m.byTokenID[rt.ID] = rt
	return nil
}
func (m *memRefresh) GetActiveByID(_ context.Context, id string) (ports.RefreshTokenRecord, bool, error) {
	rt, ok := m.byTokenID[id]
	if !ok || rt.RevokedAt != nil {
		return ports.RefreshTokenRecord{}, false, nil
	}
	return rt, true, nil
}
func (m *memRefresh) Revoke(_ context.Context, id string, when time.Time) error {
	rt, ok := m.byTokenID[id]
	if !ok {
		return nil
	}
	rt.RevokedAt = &when
	m.byTokenID[id] = rt
	return nil
}

type fakeHasher struct{}

func (h fakeHasher) HashPassword(password string) (string, error) { return "hash:" + password, nil }
func (h fakeHasher) VerifyPassword(hash string, password string) (bool, error) {
	return hash == "hash:"+password, nil
}

type fakeJWT struct{}

func (j fakeJWT) IssueAccessToken(userID string, role string) (string, error) {
	return "access:" + userID + ":" + role, nil
}

type fakeRandom struct{}

func (r fakeRandom) NewID() string { return "id-1" }
func (r fakeRandom) NewToken() (string, error) {
	return "refresh-secret", nil
}

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

type fakeVerifier struct{}

func (v fakeVerifier) VerifyAccessToken(token string) (Principal, error) {
	if token == "access:id-1:user" {
		return Principal{UserID: "id-1", Role: "user"}, nil
	}
	if token == "access:mod-1:moderator" {
		return Principal{UserID: "mod-1", Role: "moderator"}, nil
	}
	return Principal{}, ErrUnauthorized
}

type memReports struct {
	byID map[string]ports.ReportRecord
}

func (m *memReports) Create(_ context.Context, r ports.ReportRecord) error {
	m.byID[r.ID] = r
	return nil
}
func (m *memReports) GetByID(_ context.Context, id string) (ports.ReportRecord, bool, error) {
	r, ok := m.byID[id]
	return r, ok, nil
}
func (m *memReports) UpdateStatus(_ context.Context, id string, status string, when time.Time) error {
	r, ok := m.byID[id]
	if !ok {
		return ports.ErrNotFound
	}
	r.Status = status
	r.UpdatedAt = when
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

type memAttachments struct {
	byReportID map[string][]ports.AttachmentRecord
	byIdemKey  map[string]ports.AttachmentRecord
}

func (m *memAttachments) Create(_ context.Context, a ports.AttachmentRecord) error {
	if a.IdempotencyKey != "" {
		m.byIdemKey[a.ReportID+"|"+a.IdempotencyKey] = a
	}
	m.byReportID[a.ReportID] = append(m.byReportID[a.ReportID], a)
	return nil
}
func (m *memAttachments) GetByIdempotencyKey(_ context.Context, reportID string, key string) (ports.AttachmentRecord, bool, error) {
	a, ok := m.byIdemKey[reportID+"|"+key]
	return a, ok, nil
}
func (m *memAttachments) ListByReport(_ context.Context, reportID string) ([]ports.AttachmentRecord, error) {
	return append([]ports.AttachmentRecord(nil), m.byReportID[reportID]...), nil
}

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

type fakeReportRandom struct{}

func (r fakeReportRandom) NewID() string             { return "r-1" }
func (r fakeReportRandom) NewToken() (string, error) { return "unused", nil }

func TestAPI_RegisterThenMe(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	now := time.Unix(1_700_000_000, 0).UTC()

	authSvc := auth.NewService(auth.Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		AuthService:   authSvc,
		UserService:   appuser.NewService(users),
		ReportService: report.NewService(report.Deps{Reports: &memReports{byID: map[string]ports.ReportRecord{}}, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		TokenVerifier: fakeVerifier{},
	})

	// register
	body, _ := json.Marshal(map[string]any{
		"email":    "a@example.com",
		"password": "P@ssw0rd!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var reg struct {
		AccessToken    string `json:"access_token"`
		RefreshTokenID string `json:"refresh_token_id"`
		RefreshToken   string `json:"refresh_token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)
	if reg.AccessToken == "" || reg.RefreshTokenID == "" || reg.RefreshToken == "" {
		t.Fatalf("expected tokens, got %+v", reg)
	}

	// me
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req2.Header.Set("Authorization", "Bearer "+reg.AccessToken)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var me struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &me)
	if me.ID != "id-1" || me.Email != "a@example.com" || me.Role != "user" {
		t.Fatalf("unexpected me: %+v", me)
	}
}

func TestAPI_CreateReport(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	now := time.Unix(1_700_000_000, 0).UTC()

	authSvc := auth.NewService(auth.Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	reportSvc := report.NewService(report.Deps{
		Reports: reportsRepo,
		Clock:   fakeClock{t: now},
		Random:  fakeReportRandom{},
	})

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		AuthService:   authSvc,
		UserService:   appuser.NewService(users),
		ReportService: reportSvc,
		TokenVerifier: fakeVerifier{},
	})

	// register (to ensure user exists for /me, and to get access token)
	body, _ := json.Marshal(map[string]any{"email": "a@example.com", "password": "P@ssw0rd!"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var reg struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)
	if reg.AccessToken == "" {
		t.Fatalf("expected access token")
	}

	// create report
	body2, _ := json.Marshal(map[string]any{"title": "Crash", "description": "Steps..."})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+reg.AccessToken)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var created struct {
		ID     string `json:"id"`
		UserID string `json:"user_id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &created)
	if created.ID != "r-1" || created.UserID != "id-1" || created.Title != "Crash" || created.Status != "new" {
		t.Fatalf("unexpected created report: %+v", created)
	}
}

func TestAPI_ListMyReports(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	now := time.Unix(1_700_000_000, 0).UTC()

	authSvc := auth.NewService(auth.Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	reportSvc := report.NewService(report.Deps{
		Reports: reportsRepo,
		Clock:   fakeClock{t: now},
		Random:  fakeReportRandom{},
	})

	// seed reports for user id-1
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r-1", UserID: "id-1", Title: "t1", Description: "d1", Status: "new", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)})
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r-2", UserID: "id-1", Title: "t2", Description: "d2", Status: "new", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)})
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r-3", UserID: "id-1", Title: "t3", Description: "d3", Status: "new", CreatedAt: now, UpdatedAt: now})
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "x", UserID: "other", Title: "x", Description: "x", Status: "new", CreatedAt: now, UpdatedAt: now})

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		AuthService:   authSvc,
		UserService:   appuser.NewService(users),
		ReportService: reportSvc,
		TokenVerifier: fakeVerifier{},
	})

	// register -> access token
	body, _ := json.Marshal(map[string]any{"email": "a@example.com", "password": "P@ssw0rd!"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var reg struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)

	// list
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/reports?limit=2&offset=0&sort_by=created_at&sort_desc=true", nil)
	req2.Header.Set("Authorization", "Bearer "+reg.AccessToken)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &resp)
	if resp.Total != 3 {
		t.Fatalf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Items) != 2 || resp.Items[0].ID != "r-3" || resp.Items[1].ID != "r-2" {
		t.Fatalf("unexpected items: %+v", resp.Items)
	}
}

func TestAPI_GetMyReport(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	authSvc := auth.NewService(auth.Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r1", UserID: "id-1", Title: "t", Description: "d", Status: "new", CreatedAt: now, UpdatedAt: now})

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		AuthService:   authSvc,
		UserService:   appuser.NewService(users),
		ReportService: report.NewService(report.Deps{Reports: reportsRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		TokenVerifier: fakeVerifier{},
	})

	// register -> access token
	body, _ := json.Marshal(map[string]any{"email": "a@example.com", "password": "P@ssw0rd!"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var reg struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/reports/r1", nil)
	req2.Header.Set("Authorization", "Bearer "+reg.AccessToken)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var got struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &got)
	if got.ID != "r1" || got.Title != "t" {
		t.Fatalf("unexpected report: %+v", got)
	}
}

func TestAPI_ListReportAttachments(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	authSvc := auth.NewService(auth.Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r1", UserID: "id-1", Title: "t", Description: "d", Status: "new", CreatedAt: now, UpdatedAt: now})

	attsRepo := &memAttachments{byReportID: map[string][]ports.AttachmentRecord{
		"r1": {{
			ID:          "a1",
			ReportID:    "r1",
			FileName:    "x.png",
			ContentType: "image/png",
			FileSize:    10,
			StorageKey:  "tus/a1",
			CreatedAt:   now,
		}},
	}, byIdemKey: map[string]ports.AttachmentRecord{}}

	attSvc := attachment.NewService(attachment.Deps{
		Reports:      reportsRepo,
		Attachments:  attsRepo,
		Storage:      nil,
		Clock:        fakeClock{t: now},
		Random:       &fakeReportRandom{},
		MaxFileSize:  20 * 1024 * 1024,
		AllowedMIMEs: map[string]struct{}{"image/png": {}},
	})

	h := NewAPI(Deps{
		Ready:             NewReadiness(),
		AuthService:       authSvc,
		UserService:       appuser.NewService(users),
		ReportService:     report.NewService(report.Deps{Reports: reportsRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		AttachmentService: attSvc,
		AttachmentSigner:  fakeSigner{},
		TokenVerifier:     fakeVerifier{},
	})

	// register to get access token
	body, _ := json.Marshal(map[string]any{"email": "a@example.com", "password": "P@ssw0rd!"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var reg struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/reports/r1/attachments", nil)
	req2.Header.Set("Authorization", "Bearer "+reg.AccessToken)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var resp struct {
		Items []struct {
			ID          string `json:"id"`
			DownloadURL string `json:"download_url"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &resp)
	if len(resp.Items) != 1 || resp.Items[0].ID != "a1" || resp.Items[0].DownloadURL == "" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestAPI_CreateAndListMessages(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	authSvc := auth.NewService(auth.Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r1", UserID: "id-1", Title: "t", Description: "d", Status: "new", CreatedAt: now, UpdatedAt: now})

	msgRepo := &memMessages{byReportID: map[string][]ports.MessageRecord{}}
	msgSvc := message.NewService(message.Deps{Reports: reportsRepo, Messages: msgRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}})

	h := NewAPI(Deps{
		Ready:          NewReadiness(),
		AuthService:    authSvc,
		UserService:    appuser.NewService(users),
		ReportService:  report.NewService(report.Deps{Reports: reportsRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		MessageService: msgSvc,
		TokenVerifier:  fakeVerifier{},
	})

	// register to get access token
	body, _ := json.Marshal(map[string]any{"email": "a@example.com", "password": "P@ssw0rd!"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var reg struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)

	// create message
	body2, _ := json.Marshal(map[string]any{"text": "hello"})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/reports/r1/messages", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+reg.AccessToken)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr2.Code, rr2.Body.String())
	}

	// list messages
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/reports/r1/messages", nil)
	req3.Header.Set("Authorization", "Bearer "+reg.AccessToken)
	rr3 := httptest.NewRecorder()
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr3.Code, rr3.Body.String())
	}
	var resp struct {
		Items []struct {
			Text string `json:"text"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rr3.Body.Bytes(), &resp)
	if resp.Total != 1 || len(resp.Items) != 1 || resp.Items[0].Text != "hello" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

type fakeSigner struct{}

func (s fakeSigner) PresignGetObject(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://example.test/" + key, nil
}

func TestAPI_Me_UnauthorizedWithoutToken(t *testing.T) {
	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		AuthService:   nil,
		UserService:   nil,
		ReportService: nil,
		TokenVerifier: fakeVerifier{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAPI_Mod_ListReports(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r-1", UserID: "u-1", Title: "t1", Description: "d1", Status: "new", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)})
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r-2", UserID: "u-2", Title: "t2", Description: "d2", Status: "resolved", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)})

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		ReportService: report.NewService(report.Deps{Reports: reportsRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		TokenVerifier: fakeVerifier{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mod/reports?sort_desc=true", nil)
	req.Header.Set("Authorization", "Bearer access:mod-1:moderator")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 2 || len(resp.Items) != 2 {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestAPI_Mod_ChangeStatus(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r-1", UserID: "u-1", Title: "t1", Description: "d1", Status: "new", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)})

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		ReportService: report.NewService(report.Deps{Reports: reportsRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		TokenVerifier: fakeVerifier{},
	})

	body, _ := json.Marshal(map[string]any{"status": "in_review"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/mod/reports/r-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer access:mod-1:moderator")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	updated, ok := reportsRepo.byID["r-1"]
	if !ok || updated.Status != "in_review" {
		t.Fatalf("status not updated: %+v", updated)
	}
}

func TestAPI_Mod_ForbiddenForUser(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		ReportService: report.NewService(report.Deps{Reports: reportsRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		TokenVerifier: fakeVerifier{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mod/reports", nil)
	req.Header.Set("Authorization", "Bearer access:id-1:user")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAPI_Mod_GetReport(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r-1", UserID: "u-1", Title: "t1", Description: "d1", Status: "new", CreatedAt: now, UpdatedAt: now})

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		ReportService: report.NewService(report.Deps{Reports: reportsRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		TokenVerifier: fakeVerifier{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mod/reports/r-1", nil)
	req.Header.Set("Authorization", "Bearer access:mod-1:moderator")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var got struct {
		ID     string `json:"id"`
		UserID string `json:"user_id"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got.ID != "r-1" || got.UserID != "u-1" {
		t.Fatalf("unexpected report: %+v", got)
	}
}

func TestAPI_Mod_CreateAndListMessages(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reportsRepo := &memReports{byID: map[string]ports.ReportRecord{}}
	_ = reportsRepo.Create(context.Background(), ports.ReportRecord{ID: "r1", UserID: "u-1", Title: "t", Description: "d", Status: "new", CreatedAt: now, UpdatedAt: now})

	msgRepo := &memMessages{byReportID: map[string][]ports.MessageRecord{}}
	msgSvc := message.NewService(message.Deps{Reports: reportsRepo, Messages: msgRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}})

	h := NewAPI(Deps{
		Ready:          NewReadiness(),
		ReportService:  report.NewService(report.Deps{Reports: reportsRepo, Clock: fakeClock{t: now}, Random: fakeReportRandom{}}),
		MessageService: msgSvc,
		TokenVerifier:  fakeVerifier{},
	})

	// create message as moderator
	body, _ := json.Marshal(map[string]any{"text": "hello from mod"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mod/reports/r1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer access:mod-1:moderator")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// list messages as moderator
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/mod/reports/r1/messages", nil)
	req2.Header.Set("Authorization", "Bearer access:mod-1:moderator")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var resp struct {
		Items []struct {
			Text       string `json:"text"`
			SenderRole string `json:"sender_role"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &resp)
	if resp.Total != 1 || len(resp.Items) != 1 || resp.Items[0].Text != "hello from mod" || resp.Items[0].SenderRole != "moderator" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}
