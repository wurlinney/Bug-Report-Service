package attachment

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	"bug-report-service/internal/application/ports"
)

var reUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type Deps struct {
	Reports     ports.ReportRepository
	Attachments ports.AttachmentRepository
	Storage     ports.ObjectStorage
	Clock       ports.Clock
	Random      ports.Random

	MaxFileSize  int64
	AllowedMIMEs map[string]struct{}
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) Upload(ctx context.Context, req UploadRequest) (DTO, error) {
	if req.ReportID == "" {
		return DTO{}, ErrBadInput
	}
	if req.ContentType == "" || len(req.Data) == 0 {
		return DTO{}, ErrBadInput
	}
	if s.deps.MaxFileSize > 0 && int64(len(req.Data)) > s.deps.MaxFileSize {
		return DTO{}, ErrBadInput
	}
	if _, ok := s.deps.AllowedMIMEs[req.ContentType]; !ok {
		return DTO{}, ErrBadInput
	}

	if err := s.ensureReportExists(ctx, req.ReportID); err != nil {
		return DTO{}, err
	}
	// Idempotency (best-effort): return existing attachment for same report+key.
	if existing, ok, err := s.getByIdempotencyKey(ctx, req.ReportID, req.IdempotencyKey); err != nil {
		return DTO{}, err
	} else if ok {
		return toDTO(existing), nil
	}

	now := s.deps.Clock.Now()
	id := s.deps.Random.NewID()
	storageKey := buildStorageKey(req.ReportID, id, req.FileName)

	// 1) upload bytes to storage
	if err := s.deps.Storage.PutObject(ctx, storageKey, req.ContentType, req.Data); err != nil {
		return DTO{}, err
	}

	// 2) persist metadata
	rec := ports.AttachmentRecord{
		ID:             id,
		ReportID:       req.ReportID,
		FileName:       safeFileName(req.FileName),
		ContentType:    req.ContentType,
		FileSize:       int64(len(req.Data)),
		StorageKey:     storageKey,
		CreatedAt:      now,
		IdempotencyKey: req.IdempotencyKey,
	}
	if err := s.deps.Attachments.Create(ctx, rec); err != nil {
		_ = s.deps.Storage.DeleteObject(ctx, storageKey) // cleanup on DB failure
		return DTO{}, err
	}

	return toDTO(rec), nil
}

func (s *Service) Finalize(ctx context.Context, req FinalizeRequest) (DTO, error) {
	if req.ReportID == "" {
		return DTO{}, ErrBadInput
	}
	if strings.TrimSpace(req.UploadID) == "" {
		return DTO{}, ErrBadInput
	}
	if req.ContentType == "" || req.FileSize <= 0 || strings.TrimSpace(req.StorageKey) == "" {
		return DTO{}, ErrBadInput
	}
	if s.deps.MaxFileSize > 0 && req.FileSize > s.deps.MaxFileSize {
		return DTO{}, ErrBadInput
	}
	if _, ok := s.deps.AllowedMIMEs[req.ContentType]; !ok {
		return DTO{}, ErrBadInput
	}

	if err := s.ensureReportExists(ctx, req.ReportID); err != nil {
		return DTO{}, err
	}
	if existing, ok, err := s.getByIdempotencyKey(ctx, req.ReportID, req.IdempotencyKey); err != nil {
		return DTO{}, err
	} else if ok {
		return toDTO(existing), nil
	}

	now := s.deps.Clock.Now()
	// Use tus upload id as attachment id to make finalize idempotent across retries.
	id := strings.TrimSpace(req.UploadID)

	rec := ports.AttachmentRecord{
		ID:             id,
		ReportID:       req.ReportID,
		FileName:       safeFileName(req.FileName),
		ContentType:    req.ContentType,
		FileSize:       req.FileSize,
		StorageKey:     strings.TrimSpace(req.StorageKey),
		CreatedAt:      now,
		IdempotencyKey: req.IdempotencyKey,
	}
	if err := s.deps.Attachments.Create(ctx, rec); err != nil {
		return DTO{}, err
	}
	return toDTO(rec), nil
}

func (s *Service) ListForReport(ctx context.Context, req ListForReportRequest) ([]DTO, error) {
	if req.ActorRole == "" || req.ActorID == "" || req.ReportID == "" {
		return nil, ErrBadInput
	}

	_, found, err := s.deps.Reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	if req.ActorRole != "moderator" {
		return nil, ErrForbidden
	}

	items, err := s.deps.Attachments.ListByReport(ctx, req.ReportID)
	if err != nil {
		return nil, err
	}
	out := make([]DTO, 0, len(items))
	for _, a := range items {
		out = append(out, toDTO(a))
	}
	return out, nil
}

func toDTO(a ports.AttachmentRecord) DTO {
	return DTO{
		ID:          a.ID,
		ReportID:    a.ReportID,
		FileName:    a.FileName,
		ContentType: a.ContentType,
		FileSize:    a.FileSize,
		StorageKey:  a.StorageKey,
		CreatedAt:   a.CreatedAt,
	}
}

func (s *Service) ensureReportExists(ctx context.Context, reportID string) error {
	_, found, err := s.deps.Reports.GetByID(ctx, reportID)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func (s *Service) getByIdempotencyKey(ctx context.Context, reportID, idempotencyKey string) (ports.AttachmentRecord, bool, error) {
	if idempotencyKey == "" {
		return ports.AttachmentRecord{}, false, nil
	}
	return s.deps.Attachments.GetByIdempotencyKey(ctx, reportID, idempotencyKey)
}

func buildStorageKey(reportID, attachmentID, fileName string) string {
	reportID = sanitizeKeyPart(reportID)
	attachmentID = sanitizeKeyPart(attachmentID)
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" || len(ext) > 10 {
		ext = ""
	}
	return "reports/" + reportID + "/attachments/" + attachmentID + ext
}

func sanitizeKeyPart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "x"
	}
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "..", "_")
	s = reUnsafe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "._-")
	if s == "" {
		return "x"
	}
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

func safeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "file"
	}
	base = reUnsafe.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._-")
	if base == "" {
		base = "file"
	}
	if len(base) > 128 {
		base = base[:128]
	}
	return base
}
