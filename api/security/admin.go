package security

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Configuration thresholds for admin page detection
const (
	MaxAdminPages = 5 // Maximum reasonable number of admin pages
)

// AdminPageSignature defines characteristics of an admin page
type AdminPageSignature struct {
	path        string
	description string
	risk        RiskLevel
	validate    PageValidator
}

// PageValidator defines validation logic for admin pages
type PageValidator func(content []byte, headers http.Header) bool

func (s *Scanner) checkAdminPages(ctx context.Context, domain string) (*AdminPagesAnalysis, error) {
	analysis := &AdminPagesAnalysis{
		Exposed:         make([]string, 0),
		Risk:            RiskLow,
		Findings:        make([]Finding, 0),
		Recommendations: defaultRecommendations(),
		Evidence:        make(map[string]string),
		Checks:          make([]AdminCheck, 0),
	}

	adminPages := []AdminPageSignature{
		{
			path:        "/admin",
			description: "Main admin interface",
			risk:        RiskHigh,
			validate: func(content []byte, headers http.Header) bool {
				contentStr := string(content)
				return containsAdminMarkers(contentStr) ||
					containsAuthForms(contentStr)
			},
		},
		{
			path:        "/wp-admin",
			description: "WordPress admin interface",
			risk:        RiskHigh,
			validate: func(content []byte, headers http.Header) bool {
				return bytes.Contains(content, []byte("wp-admin")) ||
					bytes.Contains(content, []byte("wp-login"))
			},
		},
		{
			path:        "/administrator",
			description: "Administrative backend",
			risk:        RiskHigh,
			validate: func(content []byte, headers http.Header) bool {
				return containsAdminMarkers(string(content))
			},
		},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	totalChecked := 0
	checkState := make(map[string]bool, len(adminPages))
	for _, page := range adminPages {
		checkState[page.path] = true
	}

	for _, page := range adminPages {
		wg.Add(1)
		go func(p AdminPageSignature) {
			defer wg.Done()

			exposed, evidence := s.validateAdminPage(ctx, domain, p)
			if exposed {
				mu.Lock()
				analysis.Exposed = append(analysis.Exposed, p.path)
				analysis.Findings = append(analysis.Findings, Finding{
					Description: fmt.Sprintf("Admin interface exposed at %s: %s", p.path, p.description),
					Risk:        p.risk,
					Evidence:    evidence,
					Mitigation:  getMitigation(p.path),
				})
				analysis.Evidence[p.path] = evidence
				totalChecked++
				checkState[p.path] = false
				mu.Unlock()
			}
		}(page)
	}

	wg.Wait()

	// Apply false positive detection
	if s.detectFalsePositivesAdmin(analysis, len(adminPages), totalChecked) {
		analysis.Risk = RiskLow
		analysis.Evidence["false_positive"] = "High detection rate suggests potential false positives. Manual verification recommended."
		analysis.Checks = buildAdminChecks(adminPages, checkState)
		return analysis, nil
	}

	// Set risk level based on validated findings
	analysis.Risk = s.calculateRisk(analysis.Findings)
	analysis.Checks = buildAdminChecks(adminPages, checkState)

	return analysis, nil
}

func buildAdminChecks(pages []AdminPageSignature, state map[string]bool) []AdminCheck {
	checks := make([]AdminCheck, 0, len(pages))
	for _, page := range pages {
		passed, ok := state[page.path]
		if !ok {
			passed = true
		}
		checks = append(checks, AdminCheck{
			Path:   page.path,
			Passed: passed,
		})
	}
	return checks
}

func (s *Scanner) validateAdminPage(ctx context.Context, domain string, page AdminPageSignature) (bool, string) {
	resp, err := s.fetch(ctx, domain+page.path, requestOptions{followRedirects: boolPtr(false)})
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	// Check response code
	if resp.StatusCode == http.StatusNotFound {
		return false, ""
	}

	// Read limited content
	content, err := io.ReadAll(io.LimitReader(resp.Body, MaxContentLength))
	if err != nil {
		return false, ""
	}

	if !page.validate(content, resp.Header) {
		return false, ""
	}

	evidence := collectEvidence(page, resp.StatusCode, content)
	return true, evidence
}

func (s *Scanner) detectFalsePositivesAdmin(analysis *AdminPagesAnalysis, totalPages, pagesFound int) bool {
	if len(analysis.Exposed) > MaxAdminPages {
		return true
	}

	ratio := float64(pagesFound) / float64(totalPages)
	return ratio >= SuspiciousRatio
}

func (s *Scanner) calculateRisk(findings []Finding) RiskLevel {
	highCount := 0
	for _, finding := range findings {
		if finding.Risk >= RiskHigh {
			highCount++
		}
	}

	if highCount > 2 {
		return RiskCritical
	} else if highCount > 0 {
		return RiskHigh
	}
	return RiskLow
}

func containsAdminMarkers(content string) bool {
	markers := []string{
		"login", "admin", "dashboard", "control panel",
		"<form", "authentication", "username", "password",
	}

	lowercaseContent := strings.ToLower(content)
	for _, marker := range markers {
		if strings.Contains(lowercaseContent, marker) {
			return true
		}
	}
	return false
}

func containsAuthForms(content string) bool {
	return strings.Contains(strings.ToLower(content), "<form") &&
		(strings.Contains(strings.ToLower(content), "username") ||
			strings.Contains(strings.ToLower(content), "password"))
}

func collectEvidence(page AdminPageSignature, statusCode int, content []byte) string {
	contentPreview := string(content[:min(len(content), 200)])
	return fmt.Sprintf(
		"Path: %s\nStatus: %d\nContent Preview: %s\nValidation: Confirmed admin interface markers",
		page.path,
		statusCode,
		sanitizeContent(contentPreview),
	)
}

func getMitigation(path string) string {
	mitigations := map[string]string{
		"/wp-admin":      "Change WordPress admin URL, implement IP restrictions, enable 2FA",
		"/admin":         "Move administrative interface to a non-standard URL, implement strong access controls",
		"/administrator": "Implement IP-based access restrictions, change URL, enable 2FA",
	}

	if mitigation, ok := mitigations[path]; ok {
		return mitigation
	}
	return "Restrict access and consider changing the URL"
}

func defaultRecommendations() []string {
	return []string{
		"Implement IP-based access restrictions",
		"Use strong authentication mechanisms",
		"Enable two-factor authentication",
		"Change default admin URLs",
		"Use rate limiting on login attempts",
		"Implement security headers",
		"Enable audit logging",
		"Use HTTPS only",
	}
}

func sanitizeContent(content string) string {
	// Remove potential sensitive data
	sensitivePatterns := []string{
		`password\s*=\s*["'][^"']*["']`,
		`token\s*=\s*["'][^"']*["']`,
		`key\s*=\s*["'][^"']*["']`,
	}

	sanitized := content
	for _, pattern := range sensitivePatterns {
		sanitized = strings.ReplaceAll(sanitized, pattern, "[REDACTED]")
	}
	return sanitized
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// adminTask wires the admin page scanner into the task registry.
type adminTask struct{}

func newAdminTask() ScanTask {
	return adminTask{}
}

func (adminTask) Name() string {
	return "adminPages"
}

func (adminTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	admin, err := scanner.checkAdminPages(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.AdminPages = *admin
	return nil
}
