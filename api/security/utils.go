// utils.go
package security

import (
	"io"
	"net/http"
	"strings"
)

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

// checkRealStatus makes a full request and checks content for 404 indicators
func checkRealStatus(client *http.Client, url string) (int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Read the first 8KB of the body to check for 404 indicators
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return resp.StatusCode, err
	}
	bodyText := strings.ToLower(string(bodyBytes))

	// Check for common 404 indicators even if status code is 200
	notFoundIndicators := []string{
		"404",
		"not found",
		"page not found",
		"cannot be found",
		"doesn't exist",
		"does not exist",
		"error 404",
		"error page",
		"page missing",
	}

	for _, indicator := range notFoundIndicators {
		if strings.Contains(bodyText, indicator) {
			return http.StatusNotFound, nil
		}
	}

	return resp.StatusCode, nil
}
