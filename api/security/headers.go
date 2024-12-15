package security

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type HeaderStatus int

const (
	StatusMissing HeaderStatus = iota
	StatusMisconfigured
	StatusSuboptimal
	StatusGood
)

// HeaderRequirement defines the validation rules and recommendations for a security header
type HeaderRequirement struct {
	name        string
	description string
	risk        RiskLevel
	required    bool
	validator   func(string) (HeaderStatus, string, string) // Returns: status, issue, recommendation
}

func sortIssuesByRisk(issues []Finding) {
	riskOrder := map[RiskLevel]int{
		RiskCritical: 0,
		RiskHigh:     1,
		RiskMedium:   2,
		RiskLow:      3,
		RiskInfo:     4,
	}

	sort.SliceStable(issues, func(i, j int) bool {
		return riskOrder[issues[i].Risk] < riskOrder[issues[j].Risk]
	})
}

func (s *Scanner) checkSecurityHeaders(domain string) (*HeadersAnalysis, error) {
	resp, err := s.client.Get(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch domain: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML for title
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

	headerChecks := map[string]HeaderRequirement{
		"Strict-Transport-Security": {
			name:        "HTTP Strict Transport Security (HSTS)",
			description: "HSTS configuration",
			risk:        RiskCritical,
			required:    true,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing, "Header is missing", "Add the HSTS header with appropriate values"
				}

				directives := parseDirective(value)
				maxAge, err := strconv.ParseInt(directives["max-age"], 10, 64)
				if err != nil {
					return StatusMisconfigured, "Invalid max-age value", "Set a valid max-age value"
				}

				minRecommendedAge := time.Hour * 24 * 30 // 30 days
				if maxAge < int64(minRecommendedAge.Seconds()) {
					return StatusSuboptimal,
						fmt.Sprintf("max-age is too small (%d seconds)", maxAge),
						fmt.Sprintf("Increase max-age to at least %d seconds (30 days)", int64(minRecommendedAge.Seconds()))
				}

				if _, hasSubdomains := directives["includeSubDomains"]; !hasSubdomains {
					return StatusSuboptimal, "Missing includeSubDomains directive", "Add the includeSubDomains directive"
				}

				if _, hasPreload := directives["preload"]; !hasPreload {
					return StatusSuboptimal, "Missing preload directive", "Consider adding the preload directive"
				}

				return StatusGood, "", ""
			},
		},
		"X-Content-Type-Options": {
			name:        "X-Content-Type-Options",
			description: "MIME-type sniffing protection",
			risk:        RiskMedium,
			required:    true,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing,
						"Header is missing",
						"Add the X-Content-Type-Options header with 'nosniff' value"
				}

				value = strings.TrimSpace(strings.ToLower(value))

				if value == "nosniff" {
					return StatusGood, "", ""
				}

				if value == "" {
					return StatusMisconfigured,
						"Header is present but empty",
						"Set the header value to 'nosniff'"
				}

				return StatusMisconfigured,
					fmt.Sprintf("Invalid value '%s' - only 'nosniff' is valid", value),
					"Set the header value to 'nosniff'"
			},
		},
		"Content-Security-Policy": {
			name:        "Content Security Policy",
			description: "CSP configuration",
			risk:        RiskHigh,
			required:    true,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing, "Header is missing", "Implement a Content Security Policy"
				}

				policies := strings.Split(value, ";")
				for _, policy := range policies {
					policy = strings.TrimSpace(policy)
					if strings.Contains(policy, "unsafe-inline") {
						return StatusSuboptimal,
							"Uses unsafe-inline directive",
							"Replace unsafe-inline with nonce or hash-based values"
					}
					if strings.Contains(policy, "unsafe-eval") {
						return StatusSuboptimal,
							"Uses unsafe-eval directive",
							"Remove unsafe-eval and refactor code to avoid eval()"
					}
				}

				requiredDirectives := []string{"default-src", "script-src", "style-src"}
				for _, directive := range requiredDirectives {
					if !strings.Contains(value, directive) {
						return StatusMisconfigured,
							fmt.Sprintf("Missing %s directive", directive),
							fmt.Sprintf("Add %s directive with appropriate values", directive)
					}
				}

				return StatusGood, "", ""
			},
		},
		"X-Frame-Options": {
			name:        "X-Frame-Options",
			description: "Clickjacking protection",
			risk:        RiskMedium,
			required:    true,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing, "Header is missing", "Add X-Frame-Options header"
				}

				value = strings.ToUpper(value)
				if value != "DENY" && value != "SAMEORIGIN" {
					return StatusMisconfigured,
						"Invalid value",
						"Set value to DENY or SAMEORIGIN"
				}

				if value != "DENY" {
					return StatusSuboptimal,
						"Using SAMEORIGIN instead of DENY",
						"Consider using DENY for maximum security if frames are not needed"
				}

				return StatusGood, "", ""
			},
		},
		"Permissions-Policy": {
			name:        "Permissions Policy",
			description: "Browser features control",
			risk:        RiskMedium,
			required:    false,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing,
						"Header is missing",
						"Consider implementing Permissions-Policy to restrict browser features"
				}

				recommendedPolicies := map[string]bool{
					"geolocation": true,
					"microphone":  true,
					"camera":      true,
					"payment":     true,
				}

				policies := strings.Split(value, ",")
				for policy := range recommendedPolicies {
					found := false
					for _, p := range policies {
						if strings.Contains(p, policy) {
							found = true
							break
						}
					}
					if !found {
						return StatusSuboptimal,
							fmt.Sprintf("Missing recommended policy for %s", policy),
							fmt.Sprintf("Consider adding policy for %s", policy)
					}
				}

				return StatusGood, "", ""
			},
		},
	}

	score := 100
	highestRisk := RiskLow

	for header, check := range headerChecks {
		value := resp.Header.Get(header)
		status, issue, recommendation := check.validator(value)

		switch status {
		case StatusMissing:
			if check.required {
				score -= 20
			} else {
				score -= 10
			}
			analysis.Issues = append(analysis.Issues, Finding{
				Description: fmt.Sprintf("%s not configured", check.name),
				Risk:        check.risk,
				Evidence:    issue,
				Mitigation:  recommendation,
			})
		case StatusMisconfigured:
			score -= 15
			analysis.Issues = append(analysis.Issues, Finding{
				Description: fmt.Sprintf("%s misconfigured", check.name),
				Risk:        check.risk,
				Evidence:    issue,
				Mitigation:  recommendation,
			})
		case StatusSuboptimal:
			score -= 5
			analysis.Issues = append(analysis.Issues, Finding{
				Description: fmt.Sprintf("%s configuration suboptimal", check.name),
				Risk:        decreaseRisk(check.risk),
				Evidence:    issue,
				Mitigation:  recommendation,
			})
		case StatusGood:
			analysis.Passed = append(analysis.Passed, fmt.Sprintf("%s: %s", header, value))
		}

		if check.risk > highestRisk && status != StatusGood {
			highestRisk = check.risk
		}
	}

	sortIssuesByRisk(analysis.Issues)

	analysis.Score = calculateGrade(score)
	analysis.Risk = highestRisk

	return analysis, nil
}

func validateXContentType(value string) (HeaderStatus, []string) {
	if value == "" {
		return StatusMissing, []string{
			"Header is missing - this header prevents MIME type sniffing attacks",
		}
	}

	value = strings.TrimSpace(strings.ToLower(value))

	switch value {
	case "nosniff":
		return StatusGood, nil
	case "":
		return StatusMisconfigured, []string{
			"Header is present but empty",
		}
	default:
		return StatusMisconfigured, []string{
			fmt.Sprintf("Invalid value '%s' - only 'nosniff' is valid", value),
		}
	}
}

func formatXContentTypeError(value string, status HeaderStatus, issues []string) string {
	base := "X-Content-Type-Options"

	switch status {
	case StatusMissing:
		return fmt.Sprintf(`%s header is missing.

This header prevents browsers from MIME type sniffing, which could allow malicious files to be
interpreted as a different content type (e.g., treating a JavaScript file as an image).

Risks of missing this header:
- MIME type confusion attacks
- Malicious file execution
- Content type spoofing

Recommended value: nosniff`, base)

	case StatusMisconfigured:
		return fmt.Sprintf(`%s header is misconfigured.

Current value: %s

This header only accepts 'nosniff' as a valid value. Any other value, including
an empty value, will be ignored by browsers, leaving the site vulnerable to
MIME type sniffing attacks.

Recommended value: nosniff`, base, value)

	default:
		return ""
	}
}

// Helper function to parse header directives into a map
func parseDirective(value string) map[string]string {
	directives := make(map[string]string)
	parts := strings.Split(value, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			directives[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		} else {
			directives[strings.TrimSpace(kv[0])] = ""
		}
	}

	return directives
}

// Helper function to decrease risk level for suboptimal configurations
func decreaseRisk(risk RiskLevel) RiskLevel {
	switch risk {
	case RiskCritical:
		return RiskHigh
	case RiskHigh:
		return RiskMedium
	case RiskMedium:
		return RiskLow
	default:
		return RiskInfo
	}
}
