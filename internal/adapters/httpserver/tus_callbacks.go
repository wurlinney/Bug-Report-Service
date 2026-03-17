package httpserver

import (
	"net/http"
	"strings"

	tushandler "github.com/tus/tusd/v2/pkg/handler"
)

func TusPreUploadCreateCallback(deps Deps, maxSize int64, allowedMIMEs map[string]struct{}) func(hook tushandler.HookEvent) (tushandler.HTTPResponse, tushandler.FileInfoChanges, error) {
	return func(hook tushandler.HookEvent) (tushandler.HTTPResponse, tushandler.FileInfoChanges, error) {
		p, ok := PrincipalFromContext(hook.Context)
		if !ok || p.UserID == "" || p.Role == "" {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("unauthorized", "missing or invalid token", http.StatusUnauthorized)
		}

		if maxSize > 0 && hook.Upload.Size > maxSize {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("file_too_large", "file too large", http.StatusRequestEntityTooLarge)
		}

		meta := hook.Upload.MetaData
		reportID := strings.TrimSpace(meta["report_id"])
		filename := strings.TrimSpace(meta["filename"])
		contentType := strings.TrimSpace(meta["content_type"])
		if reportID == "" || filename == "" || contentType == "" {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("validation_error", "report_id, filename and content_type are required", http.StatusBadRequest)
		}
		if _, ok := allowedMIMEs[contentType]; !ok {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("unsupported_media_type", "unsupported content type", http.StatusBadRequest)
		}

		if deps.ReportService == nil {
			return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("misconfigured", "service misconfigured", http.StatusInternalServerError)
		}
		if p.Role != "moderator" {
			if _, err := deps.ReportService.GetForUser(hook.Context, p.UserID, reportID); err != nil {
				return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{}, tushandler.NewError("forbidden", "forbidden", http.StatusForbidden)
			}
		}

		// Persist who uploaded; and set filetype for s3store to use as Content-Type.
		newMeta := make(tushandler.MetaData, len(meta)+3)
		for k, v := range meta {
			newMeta[k] = v
		}
		newMeta["uploader_id"] = p.UserID
		newMeta["uploader_role"] = p.Role
		newMeta["filetype"] = contentType

		return tushandler.HTTPResponse{}, tushandler.FileInfoChanges{MetaData: newMeta}, nil
	}
}
