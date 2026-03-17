package report

const (
	StatusNew      = "new"
	StatusInReview = "in_review"
	StatusResolved = "resolved"
	StatusRejected = "rejected"
)

func IsValidStatus(s string) bool {
	switch s {
	case StatusNew, StatusInReview, StatusResolved, StatusRejected:
		return true
	default:
		return false
	}
}
