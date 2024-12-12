// headers.go
package security

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// headerCheck defines the validation rules for a security header
type headerCheck struct {
	description string
	risk        RiskLevel
	validator   func(string) (bool, string) // Returns: valid, reason
}

func (s *Scanner) checkSecurityHeaders(domain string) (*HeadersAnalysis, error) {
	resp, err := s.client.Get(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch domain: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var title string
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = n.FirstChild.Data
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
		}
	}
	findTitle(doc)

	analysis := &HeadersAnalysis{
		Issues: make([]Finding, 0),
		Passed: make([]string, 0),
		Risk:   RiskLow,
		Title:  title,
	}

	headerChecks := map[string]headerCheck{
		"Strict-Transport-Security": {
			description: "HSTS not properly configured",
			risk:        RiskHigh,
			validator: func(value string) (bool, string) {
				if !strings.Contains(value, "max-age=") {
					return false, "missing max-age directive"
				}
				if !strings.Contains(value, "includeSubDomains") {
					return false, "missing includeSubDomains directive"
				}
				return true, ""
			},
		},
		"Content-Security-Policy": {
			description: "Content Security Policy not properly configured",
			risk:        RiskHigh,
			validator: func(value string) (bool, string) {
				if value == "" {
					return false, "empty policy"
				}
				if strings.Contains(value, "unsafe-inline") || strings.Contains(value, "unsafe-eval") {
					return false, "contains unsafe directives"
				}
				return true, ""
			},
		},
		"X-Frame-Options": {
			description: "Clickjacking protection not properly configured",
			risk:        RiskMedium,
			validator: func(value string) (bool, string) {
				value = strings.ToUpper(value)
				if value != "DENY" && value != "SAMEORIGIN" {
					return false, "invalid value - should be DENY or SAMEORIGIN"
				}
				return true, ""
			},
		},
		"X-Content-Type-Options": {
			description: "MIME-type sniffing protection not properly configured",
			risk:        RiskMedium,
			validator: func(value string) (bool, string) {
				if value != "nosniff" {
					return false, "invalid value - must be nosniff"
				}
				return true, ""
			},
		},
		"Referrer-Policy": {
			description: "Referrer policy not properly configured",
			risk:        RiskLow,
			validator: func(value string) (bool, string) {
				validPolicies := map[string]bool{
					"no-referrer": true, "no-referrer-when-downgrade": true,
					"origin": true, "origin-when-cross-origin": true,
					"same-origin": true, "strict-origin": true,
					"strict-origin-when-cross-origin": true, "unsafe-url": true,
				}
				policies := strings.Split(value, ",")
				for _, policy := range policies {
					if !validPolicies[strings.TrimSpace(policy)] {
						return false, "contains invalid policy"
					}
				}
				return true, ""
			},
		},
		"Permissions-Policy": {
			description: "Permissions policy not properly configured",
			risk:        RiskMedium,
			validator: func(value string) (bool, string) {
				if value == "" {
					return false, "empty policy"
				}
				return true, ""
			},
		},
	}

	score := 100
	highestRisk := RiskLow

	for header, check := range headerChecks {
		value := resp.Header.Get(header)
		if value == "" {
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
			if valid, reason := check.validator(value); !valid {
				analysis.Issues = append(analysis.Issues, Finding{
					Description: check.description,
					Risk:        check.risk,
					Evidence:    fmt.Sprintf("Header '%s' has invalid value: %s", header, reason),
					Mitigation:  fmt.Sprintf("Update the '%s' header with correct values", header),
				})
				score -= 10
				if check.risk > highestRisk {
					highestRisk = check.risk
				}
			} else {
				analysis.Passed = append(analysis.Passed, fmt.Sprintf("%s: %s", header, value))
			}
		}
	}

	analysis.Score = calculateGrade(score)
	analysis.Risk = highestRisk

	return analysis, nil
}
