package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrForbidden          = errors.New("forbidden")
	ErrBadInput           = errors.New("bad input")
	ErrUniqueViolation    = errors.New("unique violation")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
)
