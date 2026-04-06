package note

import "time"

type Note struct {
	ID                string
	ReportID          string
	AuthorModeratorID string
	Text              string
	CreatedAt         time.Time
}
