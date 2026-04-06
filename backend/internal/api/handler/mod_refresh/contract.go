package mod_refresh

import (
	"context"

	uc "bug-report-service/internal/usecase/mod_refresh"
)

type UseCase interface {
	Execute(ctx context.Context, req uc.Request) (uc.Response, error)
}
