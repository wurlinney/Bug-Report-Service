package report

import "time"

type CreateRequest struct {
	UserID      string
	Title       string
	Description string
}

type ChangeStatusRequest struct {
	ActorRole string
	ReportID  string
	Status    string
}

type ListForUserRequest struct {
	ActorUserID string
	Status      *string
	Query       *string
	SortBy      string
	SortDesc    bool
	Limit       int
	Offset      int
}

type ReportDTO struct {
	ID          string
	UserID      string
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
