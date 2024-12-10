// securityfiles.go
package security

import (
	"fmt"
	"io"
	"mime"
	"strings"
	"sync"
	"time"
)

func (s *Scanner) checkRobotsTxt(domain string) (*RobotsAnalysis, error) {
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
			resp, err := s.client.Get(domain + p)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				contentType := resp.Header.Get("Content-Type")
				mediaType, _, err := mime.ParseMediaType(contentType)
				if err != nil {
					return
				}

				// Read the content
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return
				}

				mu.Lock()
				analysis.Exists = true
				analysis.ContentType = mediaType
				analysis.Content = string(bodyBytes)

				// Check if it's actually a text file
				if mediaType != "text/plain" {
					analysis.Risk = RiskMedium
					analysis.Findings = append(analysis.Findings, Finding{
						Description: "robots.txt served with incorrect content type",
						Risk:        RiskMedium,
						Evidence:    fmt.Sprintf("Content-Type: %s", contentType),
						Mitigation:  "Serve robots.txt with Content-Type: text/plain",
					})
				}

				// Check for sensitive patterns
				sensitivePatterns := []string{
					"/admin", "/api", "/internal", "/private",
					"/wp-admin", "/phpmyadmin", "/config",
				}

				contentLower := strings.ToLower(string(bodyBytes))
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
				mu.Unlock()
			}
		}(path)
	}

	wg.Wait()
	return analysis, nil
}

func (s *Scanner) checkSecurityTxt(domain string) (*SecurityTxtAnalysis, error) {
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

	for _, path := range securityPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			resp, err := s.client.Get(domain + p)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				contentType := resp.Header.Get("Content-Type")
				mediaType, _, err := mime.ParseMediaType(contentType)
				if err != nil {
					return
				}

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return
				}

				mu.Lock()
				defer mu.Unlock()

				analysis.Exists = true
				analysis.ContentType = mediaType
				analysis.Content = string(bodyBytes)

				// Check if signed
				analysis.ValidSignature = strings.Contains(analysis.Content, "-----BEGIN PGP SIGNED MESSAGE-----")

				// Parse fields
				lines := strings.Split(analysis.Content, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					switch {
					case strings.HasPrefix(line, "Contact:"):
						contact := strings.TrimPrefix(line, "Contact:")
						analysis.Contacts = append(analysis.Contacts, strings.TrimSpace(contact))
					case strings.HasPrefix(line, "Expires:"):
						expiry := strings.TrimPrefix(line, "Expires:")
						expiry = strings.TrimSpace(expiry)
						if expiryTime, err := time.Parse(time.RFC3339, expiry); err == nil {
							analysis.Expiration = expiryTime

							// Check expiration timeframes
							daysUntilExpiry := time.Until(expiryTime).Hours() / 24
							switch {
							case daysUntilExpiry < 30:
								analysis.Risk = RiskHigh
								analysis.Findings = append(analysis.Findings, Finding{
									Description: "security.txt expires in less than 30 days",
									Risk:        RiskHigh,
									Evidence:    fmt.Sprintf("Expires on: %s", expiry),
									Mitigation:  "Update security.txt with a future expiration date",
								})
							case daysUntilExpiry < 90:
								analysis.Risk = RiskMedium
								analysis.Findings = append(analysis.Findings, Finding{
									Description: "security.txt expires in less than 90 days",
									Risk:        RiskMedium,
									Evidence:    fmt.Sprintf("Expires on: %s", expiry),
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
