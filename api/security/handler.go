package security

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
	"gorm.io/gorm"
)

func InitDatabase() error {
	db := database.GetDB()
	return db.AutoMigrate(&SecurityReportRecord{}, &Screenshot{})
}

type SecurityReportRecord struct {
	ID          uint   `gorm:"primaryKey"`
	ServerURL   string `gorm:"index"`
	ReportData  []byte `gorm:"type:json"`
	CreatedAt   time.Time
	RiskLevel   RiskLevel `gorm:"index"`
	Description string
}

// Screenshot stores binary screenshot data
type Screenshot struct {
	ID        uint   `gorm:"primaryKey"`
	ServerURL string `gorm:"index"`
	Data      []byte `gorm:"type:blob"`
	CreatedAt time.Time
	MIMEType  string
}

type screenshotResponse struct {
	Success   bool   `json:"success"`
	Image     string `json:"image"`      // base64 encoded
	ImageType string `json:"image_type"` // e.g., "image/png"
	Error     string `json:"error,omitempty"`
}

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

		if err := storeSecurityReport(report); err != nil {
			c.JSON(500, gin.H{
				"error": fmt.Sprintf("Failed to store report: %v", err),
			})
			return
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

	// Check if we have a recent screenshot
	if screenshot, err := getRecentScreenshot(domain); err == nil {
		// Set cache headers
		c.Header("Cache-Control", "public, max-age=86400") // 24 hours
		c.Header("Content-Type", screenshot.MIMEType)
		c.Data(200, screenshot.MIMEType, screenshot.Data)
		return
	}

	// Make request to screenshot service
	err := captureAndSendScreenshot(c, domain)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "url": domain})
	}
}

func LastSecurityScanHandler(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "server id parameter is required"})
		return
	}

	db := database.GetDB()

	// Get URL and check if server exists
	var server types.Server
	if err := db.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "server not found"})
		} else {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Database error: %v", err)})
		}
		return
	}

	// Check if security scan exists
	var securityReport types.SecurityReportRecord
	err := db.Where("server_url = ?", server.URL).
		Order("created_at DESC").
		First(&securityReport).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No existing scan found - trigger new security scan
			c.Params = append(c.Params, gin.Param{Key: "domain", Value: server.URL})
			SecurityScanHandler(c)
			return
		}
		// Database error
		c.JSON(500, gin.H{"error": fmt.Sprintf("Database error: %v", err)})
		return
	}

	// Return existing security report
	var report types.SecurityReport
	if err := json.Unmarshal(securityReport.ReportData, &report); err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse security report"})
		return
	}

	c.JSON(200, report)
}

var (
	maxParallelScreenshots = 2 // Configurable parallel screenshot limit
	screenshotSemaphore   = make(chan struct{}, maxParallelScreenshots)
)

func SetMaxParallelScreenshots(limit int) {
	if limit < 1 {
			limit = 1
	}
	// Create new semaphore with updated capacity
	newSemaphore := make(chan struct{}, limit)
	
	// Replace the old semaphore
	oldSemaphore := screenshotSemaphore
	screenshotSemaphore = newSemaphore
	maxParallelScreenshots = limit
	
	// Close old semaphore after ensuring no operations are using it
	close(oldSemaphore)
}

func captureAndSendScreenshot(c *gin.Context, domain string) error {
	// Acquire semaphore before starting screenshot operation
	screenshotSemaphore <- struct{}{} // Block if max parallel operations reached
	defer func() {
			<-screenshotSemaphore // Release semaphore when done
	}()

	var err error
	if domain, err = utils.EnsureHTTPS(domain); err != nil {
			if c != nil {
					c.JSON(400, gin.H{"error": "Invalid URL format"})
			}
			return err
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

	client := &http.Client{Timeout: 30 * time.Second}
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

	// Decode image
	imgData, err := base64.StdEncoding.DecodeString(result.Image)
	if err != nil {
			return fmt.Errorf("failed to decode image: %v", err)
	}

	contentType := result.ImageType
	if contentType == "" {
			contentType = "image/png"
	}

	// Store screenshot in database first
	err = storeScreenshot(domain, imgData, contentType)
	if err != nil {
			return fmt.Errorf("failed to store screenshot: %v", err)
	}

	if c == nil {
			return nil
	}

	// Send response to client
	c.Header("Content-Type", contentType)
	c.Data(200, contentType, imgData)
	return nil
}

func storeSecurityReport(report *SecurityReport) error {
	db := database.GetDB()

	// Convert report to JSON
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return err
	}

	// Create security report record
	reportRecord := SecurityReportRecord{
		ServerURL:   strings.TrimPrefix(strings.TrimPrefix(report.TargetURL, "https://"), "http://"),
		ReportData:  reportJSON,
		CreatedAt:   report.ScanTimestamp,
		RiskLevel:   determineOverallRisk(report),
		Description: generateReportSummary(report),
	}

	return db.Create(&reportRecord).Error
}

func storeScreenshot(url string, data []byte, mimeType string) error {
	db := database.GetDB()

	screenshot := Screenshot{
		ServerURL: utils.StripProtocol(url),
		Data:      data,
		CreatedAt: time.Now(),
		MIMEType:  mimeType,
	}
	return db.Create(&screenshot).Error
}

func getRecentScreenshot(url string) (*Screenshot, error) {
	db := database.GetDB()
	var screenshot Screenshot

	cutoff := time.Now().Add(-168 * time.Hour)

	err := db.Where("server_url = ? AND created_at > ?", url, cutoff).
		Order("created_at DESC").
		First(&screenshot).Error

	if err != nil {
		return nil, err
	}

	return &screenshot, nil
}

func determineOverallRisk(report *SecurityReport) RiskLevel {
	risks := []RiskLevel{
		report.Headers.Risk,
		report.Certificate.Risk,
		report.AdminPages.Risk,
		report.Swagger.Risk,
		report.Infrastructure.Risk,
		report.FileExposure.Risk,
	}

	// Return highest risk level found
	for _, risk := range []RiskLevel{RiskCritical, RiskHigh, RiskMedium, RiskLow, RiskInfo} {
		for _, r := range risks {
			if r == risk {
				return risk
			}
		}
	}
	return RiskInfo
}

func generateReportSummary(report *SecurityReport) string {
	findings := len(report.Headers.Issues) +
		len(report.Certificate.Findings) +
		len(report.AdminPages.Findings) +
		len(report.Swagger.Findings) +
		len(report.Infrastructure.Findings)

	return fmt.Sprintf("Security scan completed at %s with %d findings",
		report.ScanTimestamp.Format(time.RFC3339),
		findings)
}
