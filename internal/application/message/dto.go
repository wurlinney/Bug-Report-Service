package message

import "time"

type CreateRequest struct {
	ActorRole string
	ActorID   string
	ReportID  string
	Text      string
}

type ListRequest struct {
	ActorRole string
	ActorID   string
	ReportID  string
	Limit     int
	Offset    int
	SortDesc  bool
}

type MessageDTO struct {
	ID         string
	ReportID   string
	SenderID   string
	SenderRole string
	Text       string
	CreatedAt  time.Time
}

type ListResponse struct {
	Items []MessageDTO
	Total int
}
