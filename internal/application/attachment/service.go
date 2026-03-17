package attachment

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	"bug-report-service/internal/application/policy"
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

func (s *Service) Upload(ctx context.Context, req UploadRequest) (AttachmentDTO, error) {
	if req.ActorRole == "" || req.ActorID == "" || req.ReportID == "" {
		return AttachmentDTO{}, ErrBadInput
	}
	if req.ContentType == "" || len(req.Data) == 0 {
		return AttachmentDTO{}, ErrBadInput
	}
	if s.deps.MaxFileSize > 0 && int64(len(req.Data)) > s.deps.MaxFileSize {
		return AttachmentDTO{}, ErrBadInput
	}
	if _, ok := s.deps.AllowedMIMEs[req.ContentType]; !ok {
		return AttachmentDTO{}, ErrBadInput
	}

	rep, found, err := s.deps.Reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return AttachmentDTO{}, err
	}
	if !found {
		return AttachmentDTO{}, ErrNotFound
	}
	if !policy.CanUserViewReport(req.ActorRole, req.ActorID, rep.UserID) {
		return AttachmentDTO{}, ErrForbidden
	}

	// Idempotency (best-effort): return existing attachment for same report+key.
	if req.IdempotencyKey != "" {
		if existing, ok, err := s.deps.Attachments.GetByIdempotencyKey(ctx, req.ReportID, req.IdempotencyKey); err != nil {
			return AttachmentDTO{}, err
		} else if ok {
			return toDTO(existing), nil
		}
	}

	now := s.deps.Clock.Now()
	id := s.deps.Random.NewID()
	storageKey := buildStorageKey(req.ReportID, id, req.FileName)

	// 1) upload bytes to storage
	if err := s.deps.Storage.PutObject(ctx, storageKey, req.ContentType, req.Data); err != nil {
		return AttachmentDTO{}, err
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
		UploadedByID:   req.ActorID,
		UploadedByRole: req.ActorRole,
	}
	if err := s.deps.Attachments.Create(ctx, rec); err != nil {
		_ = s.deps.Storage.DeleteObject(ctx, storageKey) // cleanup on DB failure
		return AttachmentDTO{}, err
	}

	return toDTO(rec), nil
}

func toDTO(a ports.AttachmentRecord) AttachmentDTO {
	return AttachmentDTO{
		ID:          a.ID,
		ReportID:    a.ReportID,
		FileName:    a.FileName,
		ContentType: a.ContentType,
		FileSize:    a.FileSize,
		StorageKey:  a.StorageKey,
		CreatedAt:   a.CreatedAt,
	}
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
