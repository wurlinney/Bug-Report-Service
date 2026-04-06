package report

const (
	StatusNew      = "new"
	StatusInReview = "in_review"
	StatusResolved = "resolved"
	StatusRejected = "rejected"
)

const (
	PriorityHigh   = "Высокий"
	PriorityMedium = "Средний"
	PriorityLow    = "Низкий"
	PriorityUnset  = "Не задан"
)

const (
	InfluenceBlocker = "Крит/блокер"
	InfluenceHigh    = "Высокий"
	InfluenceMedium  = "Средний"
	InfluenceLow     = "Низкий"
	InfluenceFeature = "Не баг а фича"
	InfluenceUnset   = "Не задано"
)

func IsValidStatus(s string) bool {
	switch s {
	case StatusNew, StatusInReview, StatusResolved, StatusRejected:
		return true
	default:
		return false
	}
}

func IsValidPriority(p string) bool {
	switch p {
	case PriorityHigh, PriorityMedium, PriorityLow, PriorityUnset:
		return true
	default:
		return false
	}
}

func IsValidInfluence(v string) bool {
	switch v {
	case InfluenceBlocker, "Крит/Блокер", InfluenceHigh, InfluenceMedium, InfluenceLow, InfluenceFeature, InfluenceUnset:
		return true
	default:
		return false
	}
}
