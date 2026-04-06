package delete_upload

import "context"

type AttachmentDeleter interface {
	DeleteFromSessionByStorageKey(ctx context.Context, uploadSessionID string, storageKey string) (bool, error)
}
