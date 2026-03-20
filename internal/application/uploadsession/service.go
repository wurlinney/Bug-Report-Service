package uploadsession

import (
	"context"

	"bug-report-service/internal/application/ports"
)

type Service struct {
	repo ports.UploadSessionRepository
}

func NewService(repo ports.UploadSessionRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context) (ports.UploadSessionRecord, error) {
	return s.repo.Create(ctx)
}

func (s *Service) Exists(ctx context.Context, id string) (bool, error) {
	_, found, err := s.repo.GetByID(ctx, id)
	return found, err
}
