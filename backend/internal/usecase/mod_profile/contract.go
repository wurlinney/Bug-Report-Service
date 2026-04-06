package mod_profile

import (
	"context"

	"bug-report-service/internal/domain/user"
)

type UserGetter interface {
	GetByID(ctx context.Context, id string) (user.User, bool, error)
}
