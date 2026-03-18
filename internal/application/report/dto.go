package report

import "time"

type CreateRequest struct {
	ReporterName string
	Description  string
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

	Status       *string
	ReporterName *string
	Query        *string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time

	SortBy   string
	SortDesc bool
	Limit    int
	Offset   int
}

type ReportDTO struct {
	ID           string
	ReporterName string
	Description  string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
