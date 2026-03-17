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

type ListAllRequest struct {
	ActorRole string

	Status      *string
	UserID      *string
	Query       *string
	CreatedFrom *time.Time
	CreatedTo   *time.Time

	SortBy   string
	SortDesc bool
	Limit    int
	Offset   int
}

type ReportDTO struct {
	ID          string
	UserID      string
	UserName    string
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
