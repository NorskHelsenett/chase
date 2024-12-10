// headers.go
package security

import (
	"fmt"
)

func (s *Scanner) checkSecurityHeaders(domain string) (*HeadersAnalysis, error) {
	resp, err := s.client.Get(domain)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	analysis := &HeadersAnalysis{
		Issues: make([]Finding, 0),
		Passed: make([]string, 0),
		Risk:   RiskLow,
	}

	headerChecks := map[string]struct {
		description string
		risk        RiskLevel
	}{
		"Strict-Transport-Security": {"HSTS not enabled", RiskHigh},
		"X-Frame-Options":           {"Clickjacking protection not enabled", RiskMedium},
		"X-Content-Type-Options":    {"MIME-type sniffing not prevented", RiskMedium},
		"Content-Security-Policy":   {"No content security policy configured", RiskHigh},
		"X-XSS-Protection":          {"XSS protection not enabled", RiskMedium},
		"Referrer-Policy":           {"Referrer policy not configured", RiskLow},
	}

	score := 100
	highestRisk := RiskLow

	for header, check := range headerChecks {
		if value := resp.Header.Get(header); value == "" {
			analysis.Issues = append(analysis.Issues, Finding{
				Description: check.description,
				Risk:        check.risk,
				Evidence:    fmt.Sprintf("Header '%s' is missing", header),
				Mitigation:  fmt.Sprintf("Add the '%s' header with appropriate values", header),
			})
			score -= 15
			if check.risk > highestRisk {
				highestRisk = check.risk
			}
		} else {
			analysis.Passed = append(analysis.Passed, fmt.Sprintf("%s: %s", header, value))
		}
	}

	analysis.Score = calculateGrade(score)
	analysis.Risk = highestRisk

	return analysis, nil
}
