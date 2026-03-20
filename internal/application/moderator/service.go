package moderator

import (
	"context"
	"errors"

	"bug-report-service/internal/application/ports"
)

var ErrModeratorNotFound = errors.New("moderator not found")

type Profile struct {
	ID        string
	Name      string
	Email     string
	CreatedAt int64
	UpdatedAt int64
}

type Service struct {
	users ports.UserRepository
}

func NewService(users ports.UserRepository) *Service {
	return &Service{users: users}
}

func (s *Service) GetProfile(ctx context.Context, moderatorID string) (Profile, error) {
	u, found, err := s.users.GetByID(ctx, moderatorID)
	if err != nil {
		return Profile{}, err
	}
	if !found {
		return Profile{}, ErrModeratorNotFound
	}
	return Profile{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	}, nil
}
