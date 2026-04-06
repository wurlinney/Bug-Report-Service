package tus

import "context"

type SessionChecker interface {
	Exists(ctx context.Context, id string) (bool, error)
}
