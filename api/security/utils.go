// utils.go
package security

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
