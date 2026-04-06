package create_upload_session

import (
	"context"

	"bug-report-service/internal/domain/uploadsession"
)

type UseCase struct {
	sessions SessionCreator
}

func New(sessions SessionCreator) *UseCase {
	return &UseCase{sessions: sessions}
}

func (uc *UseCase) Execute(ctx context.Context) (uploadsession.UploadSession, error) {
	return uc.sessions.Create(ctx)
}
