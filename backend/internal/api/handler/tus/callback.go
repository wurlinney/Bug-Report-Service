package tus

import (
	"context"
	"errors"
	"net/http"

	"github.com/tus/tusd/v2/pkg/handler"
)

const maxUploadSize = 50 * 1024 * 1024 // 50 MiB

// TusPreUploadCreateCallback returns a callback function suitable for use as
// tusd's PreUploadCreateCallback. It validates:
//   - upload size does not exceed the limit
//   - Upload-Metadata contains a valid upload_session_id
//   - the referenced upload session exists
func TusPreUploadCreateCallback(sessions SessionChecker) func(hook handler.HookEvent) (handler.HTTPResponse, handler.FileInfoChanges, error) {
	return func(hook handler.HookEvent) (handler.HTTPResponse, handler.FileInfoChanges, error) {
		info := hook.Upload

		if info.Size > maxUploadSize {
			return handler.HTTPResponse{
				StatusCode: http.StatusRequestEntityTooLarge,
				Body:       "file too large",
			}, handler.FileInfoChanges{}, errors.New("file too large")
		}

		sessionID := info.MetaData["upload_session_id"]
		if sessionID == "" {
			return handler.HTTPResponse{
				StatusCode: http.StatusBadRequest,
				Body:       "missing upload_session_id",
			}, handler.FileInfoChanges{}, errors.New("missing upload_session_id")
		}

		exists, err := sessions.Exists(context.Background(), sessionID)
		if err != nil {
			return handler.HTTPResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "internal error",
			}, handler.FileInfoChanges{}, err
		}
		if !exists {
			return handler.HTTPResponse{
				StatusCode: http.StatusBadRequest,
				Body:       "upload session not found",
			}, handler.FileInfoChanges{}, errors.New("upload session not found")
		}

		return handler.HTTPResponse{}, handler.FileInfoChanges{}, nil
	}
}
