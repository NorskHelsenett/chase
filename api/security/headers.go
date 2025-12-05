package security

import (
	"context"
	"fmt"
	"net/http"
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

type headersTask struct{}

func newHeadersTask() ScanTask {
	return headersTask{}
}

func (headersTask) Name() string {
	return "headers"
}

func (headersTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	analysis, err := scanner.checkSecurityHeaders(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.Headers = *analysis
	return nil
}

func (s *Scanner) checkSecurityHeaders(ctx context.Context, domain string) (*HeadersAnalysis, error) {
	resp, err := s.fetch(ctx, domain, requestOptions{})
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
		Issues:         make([]Finding, 0),
		CookieFindings: make([]Finding, 0),
		Passed:         make([]string, 0),
		Risk:           RiskLow,
		Title:          title,
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
		"Referrer-Policy": {
			name:        "Referrer Policy",
			description: "Controls how much referrer data is shared",
			risk:        RiskMedium,
			required:    true,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing, "Header is missing", "Set Referrer-Policy to strict-origin-when-cross-origin"
				}
				val := strings.ToLower(strings.TrimSpace(value))
				switch val {
				case "no-referrer", "strict-origin", "strict-origin-when-cross-origin", "same-origin":
					return StatusGood, "", ""
				case "unsafe-url":
					return StatusMisconfigured, "unsafe-url leaks full URLs", "Use strict-origin-when-cross-origin or no-referrer"
				default:
					return StatusSuboptimal, fmt.Sprintf("Policy %s is permissive", value), "Tighten the policy to strict-origin-when-cross-origin"
				}
			},
		},
		"Permissions-Policy": {
			name:        "Permissions Policy",
			description: "Restricts advanced browser features",
			risk:        RiskMedium,
			required:    false,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing, "Header is missing", "Add Permissions-Policy to disable unused sensors/APIs"
				}
				if strings.Contains(value, "*") {
					return StatusMisconfigured, "Wildcard permissions detected", "Enumerate explicit origins for each feature"
				}
				return StatusGood, "", ""
			},
		},
		"Cross-Origin-Opener-Policy": {
			name:        "Cross-Origin-Opener-Policy (COOP)",
			description: "Defends against cross-origin attacks",
			risk:        RiskMedium,
			required:    false,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing, "Header is missing", "Set COOP to same-origin"
				}
				val := strings.ToLower(value)
				if val == "same-origin" || val == "same-origin-allow-popups" {
					return StatusGood, "", ""
				}
				return StatusMisconfigured, "Unexpected COOP policy", "Prefer same-origin for isolation"
			},
		},
		"Cross-Origin-Embedder-Policy": {
			name:        "Cross-Origin-Embedder-Policy (COEP)",
			description: "Prevents loading untrusted cross-origin resources",
			risk:        RiskHigh,
			required:    false,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing, "Header is missing", "Set COEP: require-corp to enable full site isolation"
				}
				if strings.ToLower(value) != "require-corp" {
					return StatusMisconfigured, "COEP must be set to require-corp", "Use 'Cross-Origin-Embedder-Policy: require-corp'"
				}
				return StatusGood, "", ""
			},
		},
		"Cross-Origin-Resource-Policy": {
			name:        "Cross-Origin-Resource-Policy (CORP)",
			description: "Restricts how other origins load resources",
			risk:        RiskMedium,
			required:    false,
			validator: func(value string) (HeaderStatus, string, string) {
				if value == "" {
					return StatusMissing, "Header is missing", "Set CORP to same-origin or same-site"
				}
				val := strings.ToLower(value)
				if val == "cross-origin" {
					return StatusSuboptimal, "CORP allows cross-origin consumption", "Use same-origin or at least same-site"
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

	analyzeCORS(resp.Header, analysis)
	analyzeCookies(resp.Cookies(), analysis)

	sortIssuesByRisk(analysis.Issues)
	sortIssuesByRisk(analysis.CookieFindings)

	analysis.Score = calculateGrade(score)
	analysis.Risk = highestRisk

	return analysis, nil
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

func analyzeCORS(headers http.Header, analysis *HeadersAnalysis) {
	origin := headers.Get("Access-Control-Allow-Origin")
	if origin == "" || origin == "null" {
		return
	}

	credentials := strings.ToLower(headers.Get("Access-Control-Allow-Credentials"))
	if origin == "*" && strings.Contains(credentials, "true") {
		analysis.Issues = append(analysis.Issues, Finding{
			Description: "CORS allows any origin with credentials",
			Risk:        RiskHigh,
			Evidence:    "Access-Control-Allow-Origin: * with credentials",
			Mitigation:  "Return specific origins or disable credentialed requests",
		})
	}
}

func analyzeCookies(cookies []*http.Cookie, analysis *HeadersAnalysis) {
	if len(cookies) == 0 {
		analysis.Passed = append(analysis.Passed, "No cookies observed – authentication may require login to validate session settings")
		return
	}

	for _, cookie := range cookies {
		if !cookie.Secure {
			analysis.CookieFindings = append(analysis.CookieFindings, Finding{
				Description: fmt.Sprintf("Cookie %s missing Secure flag", cookie.Name),
				Risk:        RiskHigh,
				Evidence:    "Cookie transmitted over HTTP",
				Mitigation:  "Set Secure on all session cookies",
			})
		}
		if !cookie.HttpOnly {
			analysis.CookieFindings = append(analysis.CookieFindings, Finding{
				Description: fmt.Sprintf("Cookie %s missing HttpOnly flag", cookie.Name),
				Risk:        RiskHigh,
				Evidence:    "Cookie accessible to JavaScript",
				Mitigation:  "Set HttpOnly to defend against XSS",
			})
		}
		if cookie.SameSite == http.SameSiteDefaultMode {
			analysis.CookieFindings = append(analysis.CookieFindings, Finding{
				Description: fmt.Sprintf("Cookie %s missing explicit SameSite", cookie.Name),
				Risk:        RiskMedium,
				Evidence:    "SameSite default can be lax on older browsers",
				Mitigation:  "Set SameSite=Strict or Lax explicitly",
			})
		}
	}
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
