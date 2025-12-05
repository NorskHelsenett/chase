package security

import (
	"context"
	"fmt"
	"io"
	"mime"
	"strings"
	"sync"
	"time"
)

// isValidTextFile checks if the response is actually a text file
func isValidTextFile(contentType string, content string) bool {
	// Parse the media type
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	// Check if it's a text file
	if !strings.HasPrefix(mediaType, "text/") {
		return false
	}

	// Check for common HTML indicators
	contentLower := strings.ToLower(content)
	htmlIndicators := []string{
		"<!doctype", "<html", "<head", "<body",
		"<script", "<style", "<div", "<span",
	}

	for _, indicator := range htmlIndicators {
		if strings.Contains(contentLower, indicator) {
			return false
		}
	}

	return true
}

func (s *Scanner) checkRobotsTxt(ctx context.Context, domain string) (*RobotsAnalysis, error) {
	robotsPaths := []string{
		"/robots.txt",
	}

	analysis := &RobotsAnalysis{
		Exists:      false,
		ContentType: "",
		Content:     "",
		Findings:    make([]Finding, 0),
		Risk:        RiskLow,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, path := range robotsPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			resp, err := s.fetch(ctx, domain+p, requestOptions{})
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				contentType := resp.Header.Get("Content-Type")
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return
				}

				content := string(bodyBytes)

				mu.Lock()
				defer mu.Unlock()

				// Validate that it's actually a text file
				if !isValidTextFile(contentType, content) {
					analysis.Risk = RiskHigh
					analysis.Findings = append(analysis.Findings, Finding{
						Description: "robots.txt is not a valid text file",
						Risk:        RiskHigh,
						Evidence:    fmt.Sprintf("Content-Type: %s, content appears to be non-text", contentType),
						Mitigation:  "Ensure robots.txt is served as a plain text file",
					})
					return
				}

				analysis.Exists = true
				analysis.ContentType = contentType
				analysis.Content = content

				// Check for sensitive patterns
				sensitivePatterns := []string{
					"/admin", "/api", "/internal", "/private",
					"/wp-admin", "/phpmyadmin", "/config",
				}

				contentLower := strings.ToLower(content)
				for _, pattern := range sensitivePatterns {
					if strings.Contains(contentLower, pattern) {
						analysis.Risk = RiskHigh
						analysis.Findings = append(analysis.Findings, Finding{
							Description: "Sensitive path disclosed in robots.txt",
							Risk:        RiskHigh,
							Evidence:    fmt.Sprintf("Found pattern: %s", pattern),
							Mitigation:  "Remove sensitive paths from robots.txt",
						})
					}
				}
			}
		}(path)
	}

	wg.Wait()
	return analysis, nil
}

func (s *Scanner) checkSecurityTxt(ctx context.Context, domain string) (*SecurityTxtAnalysis, error) {
	securityPaths := []string{
		"/.well-known/security.txt",
		"/security.txt",
	}

	analysis := &SecurityTxtAnalysis{
		Exists:          false,
		ContentType:     "",
		Content:         "",
		ValidSignature:  false,
		Contacts:        make([]string, 0),
		Canonical:       make([]string, 0),
		Encryptions:     make([]string, 0),
		Acknowledgments: make([]string, 0),
		Findings:        make([]Finding, 0),
		Risk:            RiskLow,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var found bool

	for _, path := range securityPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			resp, err := s.fetch(ctx, domain+p, requestOptions{})
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				contentType := resp.Header.Get("Content-Type")
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return
				}

				content := string(bodyBytes)

				mu.Lock()
				defer mu.Unlock()

				// Skip if we already found and processed the file (avoid duplicates)
				if found {
					return
				}

				// Validate that it's actually a text file
				if !isValidTextFile(contentType, content) {
					analysis.Risk = RiskHigh
					analysis.Findings = append(analysis.Findings, Finding{
						Description: "security.txt is not a valid text file",
						Risk:        RiskHigh,
						Evidence:    fmt.Sprintf("Content-Type: %s, content appears to be non-text", contentType),
						Mitigation:  "Ensure security.txt is served as a plain text file",
					})
					return
				}

				found = true
				analysis.Exists = true
				analysis.ContentType = contentType
				analysis.Content = content

				// Check if signed
				analysis.ValidSignature = strings.Contains(content, "-----BEGIN PGP SIGNED MESSAGE-----")

				// Parse fields
				lines := strings.Split(content, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					switch {
					case strings.HasPrefix(line, "Contact:"):
						contact := strings.TrimPrefix(line, "Contact:")
						analysis.Contacts = append(analysis.Contacts, strings.TrimSpace(contact))
					case strings.HasPrefix(line, "Expires:"):
						expiry := strings.TrimPrefix(line, "Expires:")
						expiry = strings.TrimSpace(expiry)

						// Try multiple date formats
						var expiryTime time.Time
						var parseErr error

						// Try RFC3339 format first
						expiryTime, parseErr = time.Parse(time.RFC3339, expiry)
						if parseErr != nil {
							// Try date-only format (YYYY-MM-DD)
							expiryTime, parseErr = time.Parse("2006-01-02", expiry)
							if parseErr == nil {
								// Set to end of day for date-only format
								expiryTime = expiryTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
							}
						}

						if parseErr == nil {
							analysis.Expiration = expiryTime

							daysUntilExpiry := time.Until(expiryTime).Hours() / 24
							switch {
							case daysUntilExpiry < 0:
								// Already expired
								analysis.Risk = RiskCritical
								analysis.Findings = append(analysis.Findings, Finding{
									Description: "security.txt has expired",
									Risk:        RiskCritical,
									Evidence:    fmt.Sprintf("Expired on: %s", expiryTime.Format("2006-01-02")),
									Mitigation:  "Update security.txt with a future expiration date immediately",
								})
							case daysUntilExpiry < 7:
								// Less than 7 days
								analysis.Risk = RiskCritical
								analysis.Findings = append(analysis.Findings, Finding{
									Description: fmt.Sprintf("security.txt expires in %.0f days", daysUntilExpiry),
									Risk:        RiskCritical,
									Evidence:    fmt.Sprintf("Expires on: %s", expiryTime.Format("2006-01-02")),
									Mitigation:  "Update security.txt with a future expiration date immediately",
								})
							case daysUntilExpiry < 14:
								// Less than 14 days
								analysis.Risk = RiskHigh
								analysis.Findings = append(analysis.Findings, Finding{
									Description: fmt.Sprintf("security.txt expires in %.0f days", daysUntilExpiry),
									Risk:        RiskHigh,
									Evidence:    fmt.Sprintf("Expires on: %s", expiryTime.Format("2006-01-02")),
									Mitigation:  "Update security.txt with a future expiration date soon",
								})
							case daysUntilExpiry < 30:
								// Less than 30 days
								analysis.Risk = RiskHigh
								analysis.Findings = append(analysis.Findings, Finding{
									Description: fmt.Sprintf("security.txt expires in %.0f days", daysUntilExpiry),
									Risk:        RiskHigh,
									Evidence:    fmt.Sprintf("Expires on: %s", expiryTime.Format("2006-01-02")),
									Mitigation:  "Update security.txt with a future expiration date",
								})
							case daysUntilExpiry < 90:
								// Less than 90 days
								analysis.Risk = RiskMedium
								analysis.Findings = append(analysis.Findings, Finding{
									Description: fmt.Sprintf("security.txt expires in %.0f days", daysUntilExpiry),
									Risk:        RiskMedium,
									Evidence:    fmt.Sprintf("Expires on: %s", expiryTime.Format("2006-01-02")),
									Mitigation:  "Consider updating security.txt expiration date",
								})
							}
						}
					case strings.HasPrefix(line, "Canonical:"):
						canonical := strings.TrimPrefix(line, "Canonical:")
						analysis.Canonical = append(analysis.Canonical, strings.TrimSpace(canonical))
					case strings.HasPrefix(line, "Encryption:"):
						encryption := strings.TrimPrefix(line, "Encryption:")
						analysis.Encryptions = append(analysis.Encryptions, strings.TrimSpace(encryption))
					case strings.HasPrefix(line, "Acknowledgments:"):
						ack := strings.TrimPrefix(line, "Acknowledgments:")
						analysis.Acknowledgments = append(analysis.Acknowledgments, strings.TrimSpace(ack))
					}
				}

				// Validate required fields
				if len(analysis.Contacts) == 0 {
					analysis.Risk = RiskHigh
					analysis.Findings = append(analysis.Findings, Finding{
						Description: "security.txt missing required Contact field",
						Risk:        RiskHigh,
						Evidence:    "No Contact: field found",
						Mitigation:  "Add at least one Contact: field",
					})
				}

				if !analysis.ValidSignature {
					analysis.Findings = append(analysis.Findings, Finding{
						Description: "security.txt is not signed with PGP",
						Risk:        RiskMedium,
						Evidence:    "No PGP signature found",
						Mitigation:  "Sign the security.txt file with a PGP key",
					})
				}
			}
		}(path)
	}

	wg.Wait()
	return analysis, nil
}
