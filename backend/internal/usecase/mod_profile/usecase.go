package mod_profile

import (
	"context"

	"bug-report-service/internal/domain"
)

type Profile struct {
	ID        string
	Name      string
	Email     string
	CreatedAt int64
	UpdatedAt int64
}

type UseCase struct {
	users UserGetter
}

func New(users UserGetter) *UseCase {
	return &UseCase{users: users}
}

func (uc *UseCase) Execute(ctx context.Context, moderatorID string) (Profile, error) {
	u, found, err := uc.users.GetByID(ctx, moderatorID)
	if err != nil {
		return Profile{}, err
	}
	if !found {
		return Profile{}, domain.ErrNotFound
	}
	return Profile{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	}, nil
}
