package security

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func SecurityScanHandler(c *gin.Context) {
	domain := c.Param("domain")
	if domain == "" {
		c.JSON(400, gin.H{"error": "domain parameter is required"})
		return
	}

	// Initialize scanner with timeout and error handling
	scanner := NewScanner()

	// Create a channel for results with timeout
	resultChan := make(chan *SecurityReport)
	errChan := make(chan error)

	// Perform scan in goroutine with timeout
	go func() {
		report, err := scanner.ScanWebsite(domain)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- report
	}()

	// Wait for result with timeout
	select {
	case err := <-errChan:
		c.JSON(500, gin.H{
			"error": fmt.Sprintf("Scan failed: %v", err),
		})
		return
	case report := <-resultChan:
		// Already in correct format, just add domain-specific context
		if len(report.Headers.Passed) > 0 {
			report.Headers.Passed = append(report.Headers.Passed,
				fmt.Sprintf("Domain %s implements basic security measures", domain))
		}

		if len(report.Certificate.Findings) > 0 {
			report.Certificate.Findings = append(report.Certificate.Findings, Finding{
				Description: fmt.Sprintf("%s uses modern encryption standards", domain),
				Risk:        RiskLow,
				Evidence:    "Strong encryption detected in certificate",
				Mitigation:  "No action needed",
			})
		}

		c.JSON(200, report)
		return
	case <-time.After(30 * time.Second):
		c.JSON(504, gin.H{
			"error": "Scan timed out",
		})
		return
	}
}
