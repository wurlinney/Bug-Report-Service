package user

import (
	"context"
	"errors"

	"bug-report-service/internal/application/ports"
)

var ErrUserNotFound = errors.New("user not found")

type Profile struct {
	ID        string
	Email     string
	Role      string
	CreatedAt int64
	UpdatedAt int64
}

type Service struct {
	users ports.UserRepository
}

func NewService(users ports.UserRepository) *Service {
	return &Service{users: users}
}

func (s *Service) GetProfile(ctx context.Context, userID string) (Profile, error) {
	u, found, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if !found {
		return Profile{}, ErrUserNotFound
	}

	return Profile{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	}, nil
}
