package change_report_meta

import "context"

type ReportMetaUpdater interface {
	UpdateMeta(ctx context.Context, id string, priority string, influence string) error
}
