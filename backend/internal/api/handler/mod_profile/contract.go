package mod_profile

import (
	"context"

	uc "bug-report-service/internal/usecase/mod_profile"
)

type UseCase interface {
	Execute(ctx context.Context, moderatorID string) (uc.Profile, error)
}
