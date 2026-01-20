package security

import "strings"

func isLikelyHTML(content []byte) bool {
	lower := strings.ToLower(string(content))
	return strings.Contains(lower, "<!doctype html") ||
		strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<head") && strings.Contains(lower, "<body")
}

func calculateGrade(score int) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 85:
		return "A"
	case score >= 70:
		return "B"
	case score >= 55:
		return "C"
	case score >= 40:
		return "D"
	case score >= 20:
		return "E"
	default:
		return "F"
	}
}
