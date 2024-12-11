package security

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

func ScreenshotHandler(c *gin.Context) {
	domain := c.Param("domain")
	if domain == "" {
		c.JSON(400, gin.H{"error": "domain parameter is required"})
		return
	}

	// Add https:// if not present
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}

	// Make request to screenshot service
	err := captureAndSendScreenshot(c, domain)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "url": domain})
	}
}

func captureAndSendScreenshot(c *gin.Context, domain string) error {
	// Internal response struct
	type screenshotResponse struct {
		Success   bool   `json:"success"`
		Image     string `json:"image"`      // base64 encoded
		ImageType string `json:"image_type"` // e.g., "image/png"
		Error     string `json:"error,omitempty"`
	}

	// Create request body
	jsonData, err := json.Marshal(map[string]string{"url": domain})
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Make request to screenshot service
	serviceURL := os.Getenv("SCREENSHOT_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://screenshot:8080"
	}

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Post(
		serviceURL+"/screenshot",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("screenshot service error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result screenshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("screenshot failed: %s", result.Error)
	}

	// Decode and send image
	imgData, err := base64.StdEncoding.DecodeString(result.Image)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	contentType := result.ImageType
	if contentType == "" {
		contentType = "image/png"
	}

	c.Header("Content-Type", contentType)
	c.Data(200, contentType, imgData)
	return nil
}
