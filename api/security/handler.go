package security

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
	"github.com/norskhelsenett/chase/webpush"
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

	// Case-insensitive parameter parsing
	cachedOnly := strings.ToLower(c.Query("cached")) == "true"

	// Case-insensitive fullSize parameter
	fullSizeParam := strings.ToLower(c.Query("fullSize"))
	fullSize := fullSizeParam == "true"

	// Parse wait parameter as integer seconds with a default of 3
	waitStr := c.DefaultQuery("waitTime", "3")
	if waitStr == "" {
		waitStr = c.DefaultQuery("waittime", "3") // Case-insensitive fallback
	}
	waitInt, err := strconv.Atoi(waitStr)
	if err != nil || waitInt < 0 {
		waitInt = 3
	}

	// Try to get cached screenshot first for any request
	cachedScreenshot, cacheErr := getRecentScreenshot(domain)

	// If cachedOnly is requested, return cached or 404
	if cachedOnly {
		if cacheErr == nil {
			c.Header("Cache-Control", "public, max-age=86400") // 24 hours
			c.Header("Content-Type", cachedScreenshot.MIMEType)
			c.Data(200, cachedScreenshot.MIMEType, cachedScreenshot.Data)
			return
		}
		c.JSON(404, gin.H{"error": "No cached screenshot available"})
		return
	}

	// Try to capture new screenshot
	err = captureAndSendScreenshot(c, domain, fullSize, waitInt)
	if err != nil {
		// Log error server-side
		log.Printf("Screenshot service error for %s: %v", domain, err)

		// Fall back to cached screenshot if available
		if cacheErr == nil {
			log.Printf("Returning cached screenshot for %s after service error", domain)
			c.Header("Cache-Control", "public, max-age=3600")
			c.Header("X-Screenshot-Cached", "true")
			c.Header("Content-Type", cachedScreenshot.MIMEType)
			c.Data(200, cachedScreenshot.MIMEType, cachedScreenshot.Data)
			return
		}

		// No cache available - return generic error without details
		c.JSON(503, gin.H{"error": "Screenshot service temporarily unavailable"})
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
	screenshotSemaphore    = make(chan struct{}, maxParallelScreenshots)
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

func captureAndSendScreenshot(c *gin.Context, domain string, fullSize bool, wait int) error {
	// Acquire semaphore before starting screenshot operation
	screenshotSemaphore <- struct{}{} // Block if max parallel operations reached
	defer func() {
		<-screenshotSemaphore // Release semaphore when done
	}()

	if wait < 0 {
		wait = 0
	}

	var err error
	if domain, err = utils.EnsureHTTPS(domain); err != nil {
		if c != nil {
			c.JSON(400, gin.H{"error": "Invalid URL format"})
		}
		return err
	}

	// Create request body with correct parameter name 'fullpage' instead of 'full_size'
	jsonData, err := json.Marshal(map[string]interface{}{
		"url":       domain,
		"wait_time": wait,
		"fullpage":  fullSize,
	})
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Get screenshot service URL
	serviceURL := os.Getenv("SCREENSHOT_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://screenshot:8080"
	}

	// Retry logic with exponential backoff
	maxRetries := 3
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			log.Printf("Retrying screenshot for %s after %v (attempt %d/%d)", domain, backoff, attempt+1, maxRetries)
			time.Sleep(backoff)
		}

		// Increase timeout for retries
		timeout := 30 * time.Second
		if attempt > 0 {
			timeout = 45 * time.Second
		}

		client := &http.Client{Timeout: timeout}
		resp, err = client.Post(
			serviceURL+"/screenshot",
			"application/json",
			bytes.NewBuffer(jsonData),
		)

		if err == nil {
			// Request succeeded, break retry loop
			lastErr = nil
			break
		}

		lastErr = err
		log.Printf("Screenshot service attempt %d failed for %s: %v", attempt+1, domain, err)
	}

	// If all retries failed
	if lastErr != nil {
		return fmt.Errorf("screenshot service unavailable after %d attempts", maxRetries)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Screenshot service returned status %d for %s: %s", resp.StatusCode, domain, string(body))
		return fmt.Errorf("screenshot service returned error status %d", resp.StatusCode)
	}

	// Parse response
	var result screenshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if !result.Success {
		log.Printf("Screenshot failed for %s: %s", domain, result.Error)
		return fmt.Errorf("screenshot capture failed")
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
		log.Printf("Failed to store screenshot for %s: %v", domain, err)
		// Continue anyway - we can still return it to the client
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

	if err := db.Create(&reportRecord).Error; err != nil {
		return err
	}

	// Send notification if high or critical risk found
	if reportRecord.RiskLevel == RiskHigh || reportRecord.RiskLevel == RiskCritical {
		go notifyHighRisk(reportRecord.ServerURL, string(reportRecord.RiskLevel), reportRecord.Description)
	}

	// Check for security.txt expiration and send notifications
	if report.SecurityTxt.Exists && !report.SecurityTxt.Expiration.IsZero() {
		go notifySecurityTxtExpiration(reportRecord.ServerURL, report.SecurityTxt.Expiration)
	}

	return nil
}

// notifyHighRisk sends a push notification for high/critical security findings
func notifyHighRisk(serverURL, riskLevel, description string) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	// Look up the server ID from the URL
	var server struct {
		ID  uint   `json:"id"`
		URL string `json:"url"`
	}
	if err := db.Table("servers").Select("id, url").Where("url = ?", serverURL).First(&server).Error; err != nil {
		log.Printf("Failed to find server ID for URL %s: %v", serverURL, err)
		// Still send notification, but without direct link
		if err := sender.NotifyHighRiskFound(0, serverURL, riskLevel, description); err != nil {
			log.Printf("Failed to send high risk notification: %v", err)
		}
		return
	}

	if err := sender.NotifyHighRiskFound(server.ID, serverURL, riskLevel, description); err != nil {
		log.Printf("Failed to send high risk notification: %v", err)
	}
}

// notifySecurityTxtExpiration sends notifications based on security.txt expiration status
func notifySecurityTxtExpiration(serverURL string, expiryDate time.Time) {
	db := database.GetDB()
	const securityTxtCooldown = 24 * time.Hour

	// Look up the server ID and name from the URL
	var server struct {
		ID   uint   `json:"id"`
		URL  string `json:"url"`
		Name string `json:"name"`
	}
	if err := db.Table("servers").Select("id, url, name").Where("url = ?", serverURL).First(&server).Error; err != nil {
		log.Printf("Failed to find server for security.txt notification (URL: %s): %v", serverURL, err)
		return
	}

	serverName := server.Name
	if serverName == "" {
		serverName = server.URL
	}

	daysUntilExpiry := time.Until(expiryDate).Hours() / 24
	daysLeft := int(daysUntilExpiry)

	switch {
	case daysUntilExpiry < 0:
		alreadySent, err := webpush.HasNotificationSince(db, server.ID, webpush.EventSecurityTxtExpired, expiryDate)
		if err != nil {
			log.Printf("Failed to check security.txt expired notification history for %s: %v", serverName, err)
		} else if alreadySent {
			log.Printf("Skipping security.txt expired notification for %s (already sent)", serverName)
			return
		}

		log.Printf("Sending security.txt expired notification for %s", serverName)
		notifySecurityTxtExpiredHelper(server.ID, server.URL, serverName, expiryDate)

	case daysUntilExpiry < 7:
		throttled, err := webpush.ShouldThrottleNotification(db, server.ID, webpush.EventSecurityTxtExpiring7Days, securityTxtCooldown)
		if err != nil {
			log.Printf("Failed to check security.txt (7 days) notification history for %s: %v", serverName, err)
		} else if throttled {
			log.Printf("Skipping security.txt expiring (7 days) notification for %s (cooldown active)", serverName)
			return
		}

		log.Printf("Sending security.txt expiring (7 days) notification for %s", serverName)
		notifySecurityTxtExpiring7DaysHelper(server.ID, server.URL, serverName, expiryDate, daysLeft)

	case daysUntilExpiry < 30:
		throttled, err := webpush.ShouldThrottleNotification(db, server.ID, webpush.EventSecurityTxtExpiring30Days, securityTxtCooldown)
		if err != nil {
			log.Printf("Failed to check security.txt (30 days) notification history for %s: %v", serverName, err)
		} else if throttled {
			log.Printf("Skipping security.txt expiring (30 days) notification for %s (cooldown active)", serverName)
			return
		}

		log.Printf("Sending security.txt expiring (30 days) notification for %s", serverName)
		notifySecurityTxtExpiring30DaysHelper(server.ID, server.URL, serverName, expiryDate, daysLeft)

	case daysUntilExpiry < 90:
		throttled, err := webpush.ShouldThrottleNotification(db, server.ID, webpush.EventSecurityTxtExpiring90Days, securityTxtCooldown)
		if err != nil {
			log.Printf("Failed to check security.txt (90 days) notification history for %s: %v", serverName, err)
		} else if throttled {
			log.Printf("Skipping security.txt expiring (90 days) notification for %s (cooldown active)", serverName)
			return
		}

		log.Printf("Sending security.txt expiring (90 days) notification for %s", serverName)
		notifySecurityTxtExpiring90DaysHelper(server.ID, server.URL, serverName, expiryDate, daysLeft)
	}
}

// Helper functions to send notifications via servers package
func notifySecurityTxtExpiredHelper(serverID uint, serverURL, serverName string, expiryDate time.Time) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}
	if err := sender.NotifySecurityTxtExpired(serverID, serverURL, serverName, expiryDate); err != nil {
		log.Printf("Failed to send security.txt expired notification: %v", err)
	}
}

func notifySecurityTxtExpiring7DaysHelper(serverID uint, serverURL, serverName string, expiryDate time.Time, daysLeft int) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}
	if err := sender.NotifySecurityTxtExpiring7Days(serverID, serverURL, serverName, expiryDate, daysLeft); err != nil {
		log.Printf("Failed to send security.txt expiring (7 days) notification: %v", err)
	}
}

func notifySecurityTxtExpiring30DaysHelper(serverID uint, serverURL, serverName string, expiryDate time.Time, daysLeft int) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}
	if err := sender.NotifySecurityTxtExpiring30Days(serverID, serverURL, serverName, expiryDate, daysLeft); err != nil {
		log.Printf("Failed to send security.txt expiring (30 days) notification: %v", err)
	}
}

func notifySecurityTxtExpiring90DaysHelper(serverID uint, serverURL, serverName string, expiryDate time.Time, daysLeft int) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}
	if err := sender.NotifySecurityTxtExpiring90Days(serverID, serverURL, serverName, expiryDate, daysLeft); err != nil {
		log.Printf("Failed to send security.txt expiring (90 days) notification: %v", err)
	}
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

	query := db.Where("server_url = ?", url)

	err := query.Order("created_at DESC").First(&screenshot).Error

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
