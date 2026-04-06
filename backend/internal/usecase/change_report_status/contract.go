package change_report_status

import "context"

type ReportStatusUpdater interface {
	UpdateStatus(ctx context.Context, id string, status string) error
}
