package create_upload_session

import (
	"context"

	"bug-report-service/internal/domain/uploadsession"
)

type SessionCreator interface {
	Create(ctx context.Context) (uploadsession.UploadSession, error)
}
