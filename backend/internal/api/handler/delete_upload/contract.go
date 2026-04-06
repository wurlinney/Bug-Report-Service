package delete_upload

import "context"

type UseCase interface {
	Execute(ctx context.Context, uploadSessionID string, storageKey string) (bool, error)
}

type SessionChecker interface {
	Exists(ctx context.Context, id string) (bool, error)
}
