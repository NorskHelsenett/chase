package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Scanner performs security analysis
type Scanner struct {
	client *http.Client
}

// NewScanner creates a new security scanner
func NewScanner() *Scanner {
	// Custom transport to skip certificate verification for scanning purposes
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 10,
	}

	return &Scanner{client: client}
}

// ScanWebsite performs a complete security scan
func (s *Scanner) ScanWebsite(domain string) (*SecurityReport, error) {
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}

	var wg sync.WaitGroup
	report := &SecurityReport{}

	// Perform all checks concurrently
	wg.Add(4)

	// Check security headers
	go func() {
		defer wg.Done()
		headerAnalysis, err := s.checkSecurityHeaders(domain)
		if err == nil {
			report.Headers = *headerAnalysis
		}
	}()

	// Check SSL/TLS certificate
	go func() {
		defer wg.Done()
		certAnalysis, err := s.checkCertificate(domain)
		if err == nil {
			report.Certificate = *certAnalysis
		}
	}()

	// Check admin pages
	go func() {
		defer wg.Done()
		adminAnalysis, err := s.checkAdminPages(domain)
		if err == nil {
			report.AdminPages = *adminAnalysis
		}
	}()

	// Check Swagger/API docs
	go func() {
		defer wg.Done()
		swaggerAnalysis, err := s.checkSwaggerDocs(domain)
		if err == nil {
			report.Swagger = *swaggerAnalysis
		}
	}()

	wg.Wait()
	return report, nil
}

func (s *Scanner) checkSecurityHeaders(domain string) (*HeadersAnalysis, error) {
	resp, err := s.client.Get(domain)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	analysis := &HeadersAnalysis{
		Issues: []string{},
		Passed: []string{},
	}

	// Check security headers
	headers := map[string]string{
		"Strict-Transport-Security": "HSTS not enabled",
		"X-Frame-Options":           "X-Frame-Options header missing",
		"X-Content-Type-Options":    "X-Content-Type-Options header missing",
		"Content-Security-Policy":   "Content-Security-Policy not configured",
		"X-XSS-Protection":          "X-XSS-Protection header missing",
		"Referrer-Policy":           "Referrer-Policy not set",
	}

	score := 100
	for header, issue := range headers {
		if value := resp.Header.Get(header); value == "" {
			analysis.Issues = append(analysis.Issues, issue)
			score -= 15
		} else {
			analysis.Passed = append(analysis.Passed, fmt.Sprintf("%s header properly configured", header))
		}
	}

	// Assign grade based on score
	switch {
	case score >= 90:
		analysis.Score = "A+"
	case score >= 80:
		analysis.Score = "A"
	case score >= 70:
		analysis.Score = "B+"
	default:
		analysis.Score = "B"
	}

	return analysis, nil
}

func (s *Scanner) checkCertificate(domain string) (*CertificateAnalysis, error) {
	conn, err := tls.Dial("tcp", strings.TrimPrefix(domain, "https://")+":443", &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cert := conn.ConnectionState().PeerCertificates[0]
	analysis := &CertificateAnalysis{
		ValidUntil: cert.NotAfter.Format("2006-01-02"),
		Issuer:     cert.Issuer.CommonName,
		Findings:   []string{},
		Warnings:   []string{},
	}

	// Check certificate properties
	if time.Until(cert.NotAfter) < 30*24*time.Hour {
		analysis.Warnings = append(analysis.Warnings, "Certificate expires in less than 30 days")
	}

	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		analysis.Findings = append(analysis.Findings, "Uses strong key encryption")
	}

	// Determine grade based on certificate properties
	grade := "A"
	if len(analysis.Warnings) > 0 {
		grade = "B+"
	}
	analysis.Grade = grade

	return analysis, nil
}

func (s *Scanner) checkAdminPages(domain string) (*AdminPagesAnalysis, error) {
	commonPaths := []string{
		"/admin",
		"/wp-admin",
		"/administrator",
		"/dashboard",
		"/console",
	}

	analysis := &AdminPagesAnalysis{
		Exposed: []string{},
		Risk:    "low",
		Recommendations: []string{
			"Implement IP-based access restrictions",
			"Use strong authentication mechanisms",
			"Enable two-factor authentication",
		},
	}

	for _, path := range commonPaths {
		resp, err := s.client.Get(domain + path)
		if err != nil {
			continue
		}
		resp.Body.Close()

		// Check if page might exist based on status code
		if resp.StatusCode != 404 {
			analysis.Exposed = append(analysis.Exposed, path)
		}
	}

	// Set risk level based on findings
	if len(analysis.Exposed) > 2 {
		analysis.Risk = "high"
	} else if len(analysis.Exposed) > 0 {
		analysis.Risk = "medium"
	}

	return analysis, nil
}

func (s *Scanner) checkSwaggerDocs(domain string) (*SwaggerAnalysis, error) {
	swaggerPaths := []string{
		"/swagger",
		"/swagger-ui.html",
		"/api-docs",
		"/openapi.json",
		"/swagger/v1/swagger.json",
	}

	analysis := &SwaggerAnalysis{
		Endpoints: []string{},
		Exposed:   false,
		Risk:      "low",
		Recommendations: []string{
			"Restrict access to API documentation",
			"Implement API key authentication",
			"Use rate limiting for API endpoints",
		},
	}

	for _, path := range swaggerPaths {
		resp, err := s.client.Get(domain + path)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != 404 {
			analysis.Endpoints = append(analysis.Endpoints, path)
			analysis.Exposed = true
		}
	}

	if analysis.Exposed {
		analysis.Risk = "high"
		analysis.Recommendations = append(analysis.Recommendations,
			"Move API documentation to authenticated area",
			"Implement IP-based access restrictions")
	}

	return analysis, nil
}
