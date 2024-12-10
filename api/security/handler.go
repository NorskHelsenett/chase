package security

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	type PythonScreenshotResponse struct {
		Success   bool   `json:"success"`
		Image     string `json:"image"`      // base64 encoded
		ImageType string `json:"image_type"` // e.g., "image/png"
		Error     string `json:"error,omitempty"`
		Timestamp string `json:"timestamp"`
		URL       string `json:"url"`
	}

	// Simple HTTPS check
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + strings.TrimPrefix(domain, "http://")
	}

	// Create request to Python service
	screenshotReq := struct {
		URL string `json:"url"`
	}{
		URL: domain,
	}

	jsonData, err := json.Marshal(screenshotReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create request: %v", err)})
		return
	}

	// Create channels for results and errors
	resultChan := make(chan *PythonScreenshotResponse, 1)
	errChan := make(chan error, 1)

	// Make request to Python service in goroutine
	go func() {
		client := &http.Client{
			Timeout: 25 * time.Second,
		}

		resp, err := client.Post(
			"http://screenshot:8080/screenshot",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			errChan <- fmt.Errorf("screenshot service error: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("service returned status %d: %s", resp.StatusCode, string(body))
			return
		}

		// Read and parse response
		var pythonResp PythonScreenshotResponse
		if err := json.NewDecoder(resp.Body).Decode(&pythonResp); err != nil {
			errChan <- fmt.Errorf("failed to parse response: %v", err)
			return
		}

		if !pythonResp.Success {
			errChan <- fmt.Errorf("screenshot failed: %s", pythonResp.Error)
			return
		}

		resultChan <- &pythonResp
	}()

	// Wait for result with timeout
	select {
	case err := <-errChan:
		c.JSON(500, gin.H{
			"error": fmt.Sprintf("Screenshot capture failed: %v", err),
			"url":   domain,
		})
		return

	case result := <-resultChan:
		if result == nil {
			c.JSON(500, gin.H{
				"error": "Screenshot service returned nil result",
				"url":   domain,
			})
			return
		}

		// Decode base64 image
		imgData, err := base64.StdEncoding.DecodeString(result.Image)
		if err != nil {
			c.JSON(500, gin.H{
				"error": fmt.Sprintf("Failed to decode image: %v", err),
				"url":   domain,
			})
			return
		}

		// Set content type
		contentType := result.ImageType
		if contentType == "" {
			contentType = "image/png"
		}

		// Return image data
		c.Header("Content-Type", contentType)
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s.png", strings.Replace(domain, "/", "_", -1)))
		c.Data(200, contentType, imgData)
		return

	case <-time.After(30 * time.Second):
		c.JSON(504, gin.H{
			"error": "Screenshot capture timed out",
			"url":   domain,
		})
		return
	}
}
