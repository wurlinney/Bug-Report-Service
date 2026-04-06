package policy

// CanUserViewReport determines whether a principal with role+id
// can view a report that belongs to ownerID.
func CanUserViewReport(role string, principalID string, ownerID string) bool {
	switch role {
	case "moderator":
		return true
	case "user":
		return principalID != "" && principalID == ownerID
	default:
		return false
	}
}

func CanModeratorChangeStatus(role string) bool {
	return role == "moderator"
}
