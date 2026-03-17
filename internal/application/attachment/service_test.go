package attachment

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/application/ports"
)

type memAttachments struct {
	byReportID map[string][]ports.AttachmentRecord
	byIdemKey  map[string]ports.AttachmentRecord
	failCreate bool
}

func (m *memAttachments) Create(_ context.Context, a ports.AttachmentRecord) error {
	if m.failCreate {
		return errors.New("db error")
	}
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
func (m *memReports) UpdateStatus(_ context.Context, _ string, _ string, _ time.Time) error {
	return nil
}
func (m *memReports) ListByUser(_ context.Context, _ string, _ ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	return nil, 0, nil
}
func (m *memReports) ListAll(_ context.Context, _ ports.ReportListFilter) ([]ports.ReportRecord, int, error) {
	return nil, 0, nil
}

type memStorage struct {
	objects map[string][]byte
	deleted []string
	failPut bool
}

func (s *memStorage) PutObject(_ context.Context, key string, _ string, data []byte) error {
	if s.failPut {
		return errors.New("put failed")
	}
	if s.objects == nil {
		s.objects = map[string][]byte{}
	}
	s.objects[key] = append([]byte(nil), data...)
	return nil
}

func (s *memStorage) DeleteObject(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	delete(s.objects, key)
	return nil
}

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

type fakeRandom struct{ n int }

func (r *fakeRandom) NewID() string {
	r.n++
	return "a" + string(rune('0'+r.n))
}
func (r *fakeRandom) NewToken() (string, error) { return "unused", nil }

func TestService_Upload_ValidatesMimeAndSize(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reports := &memReports{byID: map[string]ports.ReportRecord{
		"r1": {ID: "r1", UserID: "u1", CreatedAt: now, UpdatedAt: now},
	}}
	atts := &memAttachments{byReportID: map[string][]ports.AttachmentRecord{}, byIdemKey: map[string]ports.AttachmentRecord{}}
	st := &memStorage{}

	svc := NewService(Deps{
		Reports:      reports,
		Attachments:  atts,
		Storage:      st,
		Clock:        fakeClock{t: now},
		Random:       &fakeRandom{},
		MaxFileSize:  1024,
		AllowedMIMEs: map[string]struct{}{"image/png": {}},
	})

	_, err := svc.Upload(context.Background(), UploadRequest{
		ActorRole:      "user",
		ActorID:        "u1",
		ReportID:       "r1",
		FileName:       "x.png",
		ContentType:    "image/jpeg",
		Data:           bytes.Repeat([]byte{1}, 10),
		IdempotencyKey: "k1",
	})
	if !errors.Is(err, ErrBadInput) {
		t.Fatalf("expected ErrBadInput for mime, got %v", err)
	}

	_, err = svc.Upload(context.Background(), UploadRequest{
		ActorRole:      "user",
		ActorID:        "u1",
		ReportID:       "r1",
		FileName:       "x.png",
		ContentType:    "image/png",
		Data:           bytes.Repeat([]byte{1}, 2048),
		IdempotencyKey: "k2",
	})
	if !errors.Is(err, ErrBadInput) {
		t.Fatalf("expected ErrBadInput for size, got %v", err)
	}
}

func TestService_Upload_Idempotent(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reports := &memReports{byID: map[string]ports.ReportRecord{
		"r1": {ID: "r1", UserID: "u1", CreatedAt: now, UpdatedAt: now},
	}}
	atts := &memAttachments{byReportID: map[string][]ports.AttachmentRecord{}, byIdemKey: map[string]ports.AttachmentRecord{}}
	st := &memStorage{}

	svc := NewService(Deps{
		Reports:      reports,
		Attachments:  atts,
		Storage:      st,
		Clock:        fakeClock{t: now},
		Random:       &fakeRandom{},
		MaxFileSize:  1024,
		AllowedMIMEs: map[string]struct{}{"image/png": {}},
	})

	req := UploadRequest{
		ActorRole:      "user",
		ActorID:        "u1",
		ReportID:       "r1",
		FileName:       "../x.png",
		ContentType:    "image/png",
		Data:           bytes.Repeat([]byte{1}, 10),
		IdempotencyKey: "idem-1",
	}

	first, err := svc.Upload(context.Background(), req)
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}
	second, err := svc.Upload(context.Background(), req)
	if err != nil {
		t.Fatalf("Upload#2 error: %v", err)
	}
	if first.ID != second.ID || first.StorageKey != second.StorageKey {
		t.Fatalf("expected idempotent result")
	}
}

func TestService_Upload_CleansUpStorageOnDBError(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reports := &memReports{byID: map[string]ports.ReportRecord{
		"r1": {ID: "r1", UserID: "u1", CreatedAt: now, UpdatedAt: now},
	}}
	atts := &memAttachments{byReportID: map[string][]ports.AttachmentRecord{}, byIdemKey: map[string]ports.AttachmentRecord{}, failCreate: true}
	st := &memStorage{}

	svc := NewService(Deps{
		Reports:      reports,
		Attachments:  atts,
		Storage:      st,
		Clock:        fakeClock{t: now},
		Random:       &fakeRandom{},
		MaxFileSize:  1024,
		AllowedMIMEs: map[string]struct{}{"image/png": {}},
	})

	_, err := svc.Upload(context.Background(), UploadRequest{
		ActorRole:      "user",
		ActorID:        "u1",
		ReportID:       "r1",
		FileName:       "x.png",
		ContentType:    "image/png",
		Data:           bytes.Repeat([]byte{1}, 10),
		IdempotencyKey: "k3",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(st.deleted) != 1 {
		t.Fatalf("expected storage cleanup delete, got %v", st.deleted)
	}
}

func TestService_Upload_SanitizesStorageKeyParts(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	reports := &memReports{byID: map[string]ports.ReportRecord{
		"../r1": {ID: "../r1", UserID: "u1", CreatedAt: now, UpdatedAt: now},
	}}
	atts := &memAttachments{byReportID: map[string][]ports.AttachmentRecord{}, byIdemKey: map[string]ports.AttachmentRecord{}}
	st := &memStorage{}

	svc := NewService(Deps{
		Reports:      reports,
		Attachments:  atts,
		Storage:      st,
		Clock:        fakeClock{t: now},
		Random:       &fakeRandom{},
		MaxFileSize:  1024,
		AllowedMIMEs: map[string]struct{}{"image/png": {}},
	})

	got, err := svc.Upload(context.Background(), UploadRequest{
		ActorRole:      "user",
		ActorID:        "u1",
		ReportID:       "../r1",
		FileName:       "x.png",
		ContentType:    "image/png",
		Data:           bytes.Repeat([]byte{1}, 10),
		IdempotencyKey: "k4",
	})
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}
	if bytes.Contains([]byte(got.StorageKey), []byte("..")) {
		t.Fatalf("expected sanitized storage key, got %q", got.StorageKey)
	}
	if bytes.Contains([]byte(got.StorageKey), []byte("\\\\")) {
		t.Fatalf("expected no backslashes, got %q", got.StorageKey)
	}
}
