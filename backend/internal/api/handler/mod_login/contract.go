package mod_login

import (
	"context"

	uc "bug-report-service/internal/usecase/mod_login"
)

type UseCase interface {
	Execute(ctx context.Context, req uc.Request) (uc.Response, error)
}
