package report

import "errors"

var (
	ErrNotFound  = errors.New("report not found")
	ErrForbidden = errors.New("forbidden")
	ErrBadInput  = errors.New("bad input")
)
