package create_upload_session

import (
	"context"

	"bug-report-service/internal/domain/uploadsession"
)

type UseCase interface {
	Execute(ctx context.Context) (uploadsession.UploadSession, error)
}
