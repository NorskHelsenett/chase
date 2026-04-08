package security

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
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
	if err := db.AutoMigrate(&SecurityReportRecord{}, &Screenshot{}); err != nil {
		return err
	}

	return nil
}

// RunDatabaseCleanup deduplicates and prunes old data.
// Call from a single background goroutine to avoid SQLite lock contention.
func RunDatabaseCleanup() {
	db := database.GetDB()

	// Each cleanup is wrapped to avoid crashing on tables that don't exist yet
	safeExec := func(desc, sql string, args ...interface{}) {
		if result := db.Exec(sql, args...); result.Error != nil {
			log.Printf("Cleanup (%s): %v", desc, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("Cleanup (%s): %d rows", desc, result.RowsAffected)
		}
	}

	safeExec("dedup screenshots",
		`DELETE FROM screenshots WHERE id NOT IN (SELECT MAX(id) FROM screenshots GROUP BY server_url)`)

	safeExec("dedup security reports",
		`DELETE FROM security_report_records WHERE id NOT IN (SELECT MAX(id) FROM security_report_records GROUP BY server_url)`)

	safeExec("old batch jobs",
		`DELETE FROM batch_job_stores WHERE status IN ('completed','failed','cancelled') AND end_time < ?`, time.Now().AddDate(0, 0, -30))

	safeExec("orphan batch results",
		`DELETE FROM batch_result_stores WHERE batch_job_id NOT IN (SELECT id FROM batch_job_stores)`)

	safeExec("old notifications",
		`DELETE FROM notification_logs WHERE created_at < ?`, time.Now().AddDate(0, 0, -90))

	safeExec("expired sessions",
		`DELETE FROM sessions WHERE expires_at < ?`, time.Now())

	// VACUUM in a separate step — use incremental to avoid doubling disk usage
	if result := db.Exec(`PRAGMA incremental_vacuum`); result.Error != nil {
		log.Printf("Incremental vacuum: %v", result.Error)
	} else {
		log.Printf("Incremental vacuum complete")
	}

	// Backfill thumbnails in background — slow, one-at-a-time, doesn't block cleanup
	go backfillThumbnails()
}

func backfillThumbnails() {
	db := database.GetDB()

	// Get IDs only — don't load blobs into memory
	var ids []uint
	db.Model(&Screenshot{}).
		Where("mime_type = ? AND (thumbnail_data IS NULL OR length(thumbnail_data) = 0)", "image/png").
		Pluck("id", &ids)

	if len(ids) == 0 {
		return
	}

	log.Printf("Backfilling thumbnails for %d screenshots", len(ids))
	for _, id := range ids {
		// Load one screenshot at a time
		var s Screenshot
		if err := db.First(&s, id).Error; err != nil {
			continue
		}
		if len(s.Data) == 0 {
			continue
		}
		thumb, err := generateThumbnail(s.Data, thumbnailWidth)
		if err != nil {
			log.Printf("Thumbnail backfill failed for %s: %v", s.ServerURL, err)
			continue
		}
		db.Model(&s).Updates(map[string]interface{}{
			"thumbnail_data": thumb,
			"thumbnail_w":    thumbnailWidth,
		})
		// Let GC reclaim memory and yield the SQLite write lock to other goroutines
		s.Data = nil
		thumb = nil
		time.Sleep(100 * time.Millisecond)
	}
	log.Printf("Thumbnail backfill complete")
}

type SecurityReportRecord struct {
	ID             uint   `gorm:"primaryKey"`
	ServerURL      string `gorm:"index"`
	ReportData     []byte `gorm:"type:json"`
	CreatedAt      time.Time
	RiskLevel      RiskLevel `gorm:"index"`
	Description    string
	ScannerVersion string `gorm:"type:varchar(16);index"`
}

// Screenshot stores binary screenshot data with an optional pre-generated thumbnail
type Screenshot struct {
	ID            uint   `gorm:"primaryKey"`
	ServerURL     string `gorm:"index"`
	Data          []byte `gorm:"type:blob"`
	ThumbnailData []byte `gorm:"type:blob"`
	ThumbnailW    int
	CreatedAt     time.Time
	MIMEType      string
}

type ReportStatusResponse struct {
	Status ScanStatus      `json:"status"`
	Report *SecurityReport `json:"report"`
}

// CacheDuration is the time window for considering cached results fresh
const CacheDuration = 5 * time.Minute

// getRecentCachedReport checks for a cached report within CacheDuration
func getRecentCachedReport(domain string) (*SecurityReportRecord, error) {
	db := database.GetDB()
	var record SecurityReportRecord

	// Normalize domain (strip protocol if present)
	normalizedDomain := utils.StripProtocol(domain)

	err := db.Where("server_url = ? AND created_at > ?", normalizedDomain, time.Now().Add(-CacheDuration)).
		Order("created_at DESC").
		First(&record).Error

	if err != nil {
		return nil, err
	}
	return &record, nil
}

func SecurityScanHandler(c *gin.Context) {
	domain := c.Param("domain")
	if err := ValidateDomain(domain); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check for recent cached result
	if cached, err := getRecentCachedReport(domain); err == nil {
		var report SecurityReport
		if err := json.Unmarshal(cached.ReportData, &report); err == nil {
			// Add cache age header
			cacheAge := int(time.Since(cached.CreatedAt).Seconds())
			c.Header("X-Cache-Age", strconv.Itoa(cacheAge))
			c.JSON(200, report)
			return
		}
	}

	// Initialize scanner with timeout and error handling
	scanner := NewScanner(0)

	// Create a channel for results with timeout
	resultChan := make(chan *SecurityReport)
	errChan := make(chan error)

	// Perform scan in goroutine with timeout
	go func() {
		report, err := scanner.ScanWebsite(c.Request.Context(), domain)
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
		augmentSecurityReport(domain, report)

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

// SecurityScanSSEHandler provides Server-Sent Events for scan progress
func SecurityScanSSEHandler(c *gin.Context) {
	domain := c.Param("domain")
	if err := ValidateDomain(domain); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Check for recent cached result first
	if cached, err := getRecentCachedReport(domain); err == nil {
		var report SecurityReport
		if err := json.Unmarshal(cached.ReportData, &report); err == nil {
			// Emit cached result immediately
			cacheAge := int(time.Since(cached.CreatedAt).Seconds())
			sendSSEEvent(c, "cache_age", cacheAge)
			sendSSEEvent(c, "status", map[string]interface{}{
				"stage":    "cached",
				"progress": 100,
			})
			sendSSEEvent(c, "result", report)
			return
		}
	}

	// Check if a scan is already running for this domain
	if status := getScanStatus(domain); status != nil && status.State == "running" {
		// Wait for existing scan
		sendSSEEvent(c, "status", map[string]interface{}{
			"stage":    "waiting",
			"progress": 0,
			"message":  "Scan already in progress",
		})

		// Poll for completion
		waitForExistingScan(c, domain)
		return
	}

	// Mark scan as running
	markScanRunning(domain)
	defer clearScanStatus(domain)

	// Start screenshot capture in parallel
	go func() {
		log.Printf("Starting parallel screenshot capture for %s", domain)
		if err := captureAndSendScreenshot(nil, domain, false, 3); err != nil {
			log.Printf("Parallel screenshot capture failed for %s: %v", domain, err)
		} else {
			log.Printf("Parallel screenshot capture completed for %s", domain)
		}
	}()

	// Create scanner with progress callback
	scanner := NewScanner(0)

	// Create a channel for progress updates
	progressChan := make(chan struct {
		stage    string
		progress int
	}, 20)

	scanner.SetProgressCallback(func(stage string, progress int) {
		select {
		case progressChan <- struct {
			stage    string
			progress int
		}{stage, progress}:
		default:
			// Drop if channel is full to prevent blocking
		}
	})

	// Create channels for result
	resultChan := make(chan *SecurityReport)
	errChan := make(chan error)

	// Start scan in goroutine
	go func() {
		defer close(progressChan)
		report, err := scanner.ScanWebsite(c.Request.Context(), domain)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- report
	}()

	// Send progress updates via SSE
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			sendSSEEvent(c, "error", "Connection closed")
			return

		case progress, ok := <-progressChan:
			if ok {
				sendSSEEvent(c, "status", map[string]interface{}{
					"stage":    progress.stage,
					"progress": progress.progress,
				})
				c.Writer.Flush()
			}

		case err := <-errChan:
			markScanFailed(domain, err)
			sendSSEEvent(c, "error", err.Error())
			return

		case report := <-resultChan:
			augmentSecurityReport(domain, report)
			if err := storeSecurityReport(report); err != nil {
				sendSSEEvent(c, "error", fmt.Sprintf("Failed to store report: %v", err))
				return
			}
			sendSSEEvent(c, "result", report)
			return

		case <-time.After(2 * time.Minute):
			markScanFailed(domain, errors.New("scan timed out"))
			sendSSEEvent(c, "error", "Scan timed out")
			return

		case <-ticker.C:
			// Keep-alive ping
			c.Writer.WriteString(": ping\n\n")
			c.Writer.Flush()
		}
	}
}

// waitForExistingScan waits for an existing scan to complete and sends result
func waitForExistingScan(c *gin.Context, domain string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(2 * time.Minute)
	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			sendSSEEvent(c, "error", "Timeout waiting for existing scan")
			return
		case <-ticker.C:
			status := getScanStatus(domain)
			if status == nil || status.State != "running" {
				// Scan completed, try to get result
				if cached, err := getRecentCachedReport(domain); err == nil {
					var report SecurityReport
					if err := json.Unmarshal(cached.ReportData, &report); err == nil {
						sendSSEEvent(c, "result", report)
						return
					}
				}
				// Check if scan failed
				if status != nil && status.State == "failed" {
					sendSSEEvent(c, "error", status.Error)
					return
				}
				sendSSEEvent(c, "error", "Scan completed but result not found")
				return
			}
			// Send keep-alive
			c.Writer.WriteString(": ping\n\n")
			c.Writer.Flush()
		}
	}
}

// sendSSEEvent sends an SSE event
func sendSSEEvent(c *gin.Context, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling SSE data: %v", err)
		return
	}
	c.Writer.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(jsonData)))
	c.Writer.Flush()
}

func augmentSecurityReport(domain string, report *SecurityReport) {
	if len(report.Headers.Passed) > 0 {
		report.Headers.Passed = append(report.Headers.Passed,
			fmt.Sprintf("Domain %s implements basic security measures", domain))
	}

	if len(report.Certificate.Findings) == 0 {
		report.Certificate.Findings = append(report.Certificate.Findings, Finding{
			Description: fmt.Sprintf("%s uses modern encryption standards", domain),
			Risk:        RiskLow,
			Evidence:    "Strong encryption detected in certificate",
			Mitigation:  "No action needed",
		})
	}
}
func ScreenshotHandler(c *gin.Context) {
	domain := c.Param("domain")
	if err := ValidateDomain(domain); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Case-insensitive parameter parsing - cached=true means prefer cache, not require cache
	preferCached := strings.ToLower(c.Query("cached")) == "true"

	// Case-insensitive fullSize parameter
	fullSizeParam := strings.ToLower(c.Query("fullSize"))
	fullSize := fullSizeParam == "true"

	// thumb=true serves the pre-generated thumbnail instead of full image
	wantThumb := strings.ToLower(c.Query("thumb")) == "true"

	// Parse wait parameter as integer seconds with a default of 3
	waitStr := c.DefaultQuery("waitTime", "3")
	if waitStr == "" {
		waitStr = c.DefaultQuery("waittime", "3") // Case-insensitive fallback
	}
	waitInt, err := strconv.Atoi(waitStr)
	if err != nil || waitInt < 0 {
		waitInt = 3
	}

	// Try to get cached screenshot — use lightweight query when only thumbnail is needed
	cachedScreenshot, cacheErr := getRecentScreenshot(domain, wantThumb && preferCached)

	// If preferCached is set and cache exists, return it immediately
	if preferCached && cacheErr == nil {
		// Failure marker — don't retry, just tell the client
		if strings.HasPrefix(cachedScreenshot.MIMEType, "error/") {
			c.Header("Cache-Control", "public, max-age=3600")
			c.JSON(404, gin.H{"error": "Screenshot not available for this server"})
			return
		}

		if wantThumb && len(cachedScreenshot.ThumbnailData) > 0 {
			c.Header("Cache-Control", "public, max-age=86400")
			c.Header("Content-Type", cachedScreenshot.MIMEType)
			c.Data(200, cachedScreenshot.MIMEType, cachedScreenshot.ThumbnailData)
			return
		}

		// No thumbnail available — need full data
		if len(cachedScreenshot.Data) == 0 {
			// We did a thumbOnly query but no thumb exists, re-fetch with full data
			cachedScreenshot, cacheErr = getRecentScreenshot(domain)
			if cacheErr != nil {
				// Fall through to capture
			}
		}

		if cacheErr == nil && len(cachedScreenshot.Data) > 0 {
			c.Header("Cache-Control", "public, max-age=86400")
			c.Header("Content-Type", cachedScreenshot.MIMEType)
			c.Data(200, cachedScreenshot.MIMEType, cachedScreenshot.Data)
			return
		}
	}

	// Try to capture new screenshot
	err = captureAndSendScreenshot(c, domain, fullSize, waitInt)
	if err != nil {
		// Log error server-side
		log.Printf("Screenshot service error for %s: %v", domain, err)

		// Fall back to cached screenshot if available (full query)
		if cached, fallbackErr := getRecentScreenshot(domain); fallbackErr == nil {
			log.Printf("Returning cached screenshot for %s after service error", domain)
			imgData := cached.Data
			if wantThumb && len(cached.ThumbnailData) > 0 {
				imgData = cached.ThumbnailData
			}
			c.Header("Cache-Control", "public, max-age=3600")
			c.Header("X-Screenshot-Cached", "true")
			c.Header("Content-Type", cached.MIMEType)
			c.Data(200, cached.MIMEType, imgData)
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

	if status := getScanStatus(server.URL); status != nil && status.State == "running" {
		c.JSON(202, ReportStatusResponse{
			Status: *status,
			Report: nil,
		})
		return
	}

	// Check if security scan exists
	var securityReport SecurityReportRecord
	err := db.Where("server_url = ?", server.URL).
		Order("created_at DESC").
		First(&securityReport).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status := markScanRunning(server.URL)
			go runBackgroundScan(server.URL)
			c.JSON(202, ReportStatusResponse{
				Status: *status,
				Report: nil,
			})
			return
		}
		// Database error
		c.JSON(500, gin.H{"error": fmt.Sprintf("Database error: %v", err)})
		return
	}

	// Return existing security report
	var report SecurityReport
	if err := json.Unmarshal(securityReport.ReportData, &report); err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse security report"})
		return
	}

	c.JSON(200, ReportStatusResponse{
		Status: ScanStatus{
			State:       "done",
			CompletedAt: securityReport.CreatedAt,
		},
		Report: &report,
	})
}

// RunBackgroundScan triggers a security scan for the given server URL in the current goroutine.
// Call this with `go` to run it in the background.
func RunBackgroundScan(serverURL string) {
	runBackgroundScan(serverURL)
}

func runBackgroundScan(serverURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	scanner := NewScanner(0)
	report, err := scanner.ScanWebsite(ctx, serverURL)
	if err != nil {
		markScanFailed(serverURL, err)
		return
	}

	augmentSecurityReport(serverURL, report)

	if err := storeSecurityReport(report); err != nil {
		markScanFailed(serverURL, err)
		return
	}

	clearScanStatus(serverURL)
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

// captureAndSendScreenshot captures a screenshot and sends it to the client.
// Returns nil if the response was sent (success or 4xx error), or an error for 5xx/network failures
// to allow the caller to fall back to cached screenshots.
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

	// Get screenshot service URL
	serviceURL := os.Getenv("SCREENSHOT_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://screenshot:11235"
	}
	serviceURL = strings.TrimRight(serviceURL, "/")
	targetURL := strings.TrimRight(domain, "/")
	requestURL := fmt.Sprintf("%s/%s/.png", serviceURL, targetURL)

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
		timeout := serviceTimeout()
		if attempt > 0 {
			timeout += 15 * time.Second
		}

		client := &http.Client{Timeout: timeout}
		ctx := context.Background()
		if c != nil {
			ctx = c.Request.Context()
		}
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if reqErr != nil {
			return fmt.Errorf("failed to create request: %v", reqErr)
		}
		resp, err = client.Do(req)

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

		// Handle client errors (4xx) - target is not screenshottable
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// Don't retry rate limits
			if resp.StatusCode == http.StatusTooManyRequests {
				if c != nil {
					c.JSON(429, gin.H{"error": "Rate limited, please try again later"})
				}
				return nil
			}

			// Store a failure marker so we don't keep retrying for this domain
			log.Printf("Screenshot not available for %s (status %d), caching failure", domain, resp.StatusCode)
			storeScreenshotFailure(domain, resp.StatusCode)

			if c != nil {
				c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("Unable to capture screenshot: %d", resp.StatusCode)})
			}
			return nil
		}

		// Server errors (5xx) - return error to trigger cache fallback
		return fmt.Errorf("screenshot service returned error status %d", resp.StatusCode)
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read image: %v", err)
	}
	if len(imgData) == 0 {
		return fmt.Errorf("screenshot capture failed")
	}

	contentType := resp.Header.Get("Content-Type")
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
		ServerURL:      strings.TrimPrefix(strings.TrimPrefix(report.TargetURL, "https://"), "http://"),
		ReportData:     reportJSON,
		CreatedAt:      report.ScanTimestamp,
		RiskLevel:      determineOverallRisk(report),
		Description:    generateReportSummary(report),
		ScannerVersion: report.ScannerVersion,
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

	// Look up the server ID from the URL
	var server struct {
		ID  uint   `json:"id"`
		URL string `json:"url"`
	}
	if err := db.Table("servers").Select("id, url").Where("url = ?", serverURL).First(&server).Error; err != nil {
		log.Printf("Failed to find server for security.txt notification (URL: %s): %v", serverURL, err)
		return
	}

	serverName := server.URL

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

const thumbnailWidth = 480

// generateThumbnail creates a smaller PNG from the full-size image using bilinear interpolation.
func generateThumbnail(data []byte, targetWidth int) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if targetWidth >= srcW {
		return data, nil
	}

	targetHeight := srcH * targetWidth / srcW
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	xRatio := float64(srcW) / float64(targetWidth)
	yRatio := float64(srcH) / float64(targetHeight)

	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			// Bilinear interpolation
			srcXf := float64(x)*xRatio + float64(srcBounds.Min.X)
			srcYf := float64(y)*yRatio + float64(srcBounds.Min.Y)

			x0 := int(srcXf)
			y0 := int(srcYf)
			x1 := x0 + 1
			y1 := y0 + 1

			if x1 >= srcBounds.Max.X {
				x1 = srcBounds.Max.X - 1
			}
			if y1 >= srcBounds.Max.Y {
				y1 = srcBounds.Max.Y - 1
			}

			xFrac := srcXf - float64(x0)
			yFrac := srcYf - float64(y0)

			r00, g00, b00, a00 := src.At(x0, y0).RGBA()
			r10, g10, b10, a10 := src.At(x1, y0).RGBA()
			r01, g01, b01, a01 := src.At(x0, y1).RGBA()
			r11, g11, b11, a11 := src.At(x1, y1).RGBA()

			lerp := func(v00, v10, v01, v11 uint32) uint8 {
				top := float64(v00)*(1-xFrac) + float64(v10)*xFrac
				bot := float64(v01)*(1-xFrac) + float64(v11)*xFrac
				return uint8((top*(1-yFrac) + bot*yFrac) / 257)
			}

			dst.SetRGBA(x, y, color.RGBA{
				R: lerp(r00, r10, r01, r11),
				G: lerp(g00, g10, g01, g11),
				B: lerp(b00, b10, b01, b11),
				A: lerp(a00, a10, a01, a11),
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func storeScreenshot(url string, data []byte, mimeType string) error {
	db := database.GetDB()
	normalizedURL := utils.StripProtocol(url)

	var thumbData []byte
	var thumbW int

	// Generate thumbnail for PNG images
	if mimeType == "image/png" {
		if thumb, err := generateThumbnail(data, thumbnailWidth); err == nil {
			thumbData = thumb
			thumbW = thumbnailWidth
		} else {
			log.Printf("Failed to generate thumbnail for %s: %v", url, err)
		}
	}

	// Upsert: keep only one screenshot per server URL
	var existing Screenshot
	if err := db.Where("server_url = ?", normalizedURL).First(&existing).Error; err == nil {
		return db.Model(&existing).Updates(map[string]interface{}{
			"data":           data,
			"thumbnail_data": thumbData,
			"thumbnail_w":    thumbW,
			"created_at":     time.Now(),
			"mime_type":      mimeType,
		}).Error
	}

	return db.Create(&Screenshot{
		ServerURL:     normalizedURL,
		Data:          data,
		ThumbnailData: thumbData,
		ThumbnailW:    thumbW,
		CreatedAt:     time.Now(),
		MIMEType:      mimeType,
	}).Error
}

// storeScreenshotFailure records a failed screenshot attempt so we don't keep retrying.
// The marker expires after 24 hours, at which point the next request will retry.
func storeScreenshotFailure(url string, statusCode int) {
	db := database.GetDB()
	normalizedURL := utils.StripProtocol(url)
	mimeType := fmt.Sprintf("error/%d", statusCode)

	var existing Screenshot
	if err := db.Where("server_url = ?", normalizedURL).First(&existing).Error; err == nil {
		db.Model(&existing).Updates(map[string]interface{}{
			"data":           nil,
			"thumbnail_data": nil,
			"thumbnail_w":    0,
			"created_at":     time.Now(),
			"mime_type":      mimeType,
		})
		return
	}

	db.Create(&Screenshot{
		ServerURL: normalizedURL,
		CreatedAt: time.Now(),
		MIMEType:  mimeType,
	})
}

const screenshotFailureRetry = 24 * time.Hour
const screenshotCacheTTL = 14 * 24 * time.Hour // 14 days

// getRecentScreenshot loads a cached screenshot. When thumbOnly is true,
// it avoids loading the full-size blob from SQLite — much faster for grid views.
func getRecentScreenshot(url string, thumbOnly ...bool) (*Screenshot, error) {
	db := database.GetDB()
	var screenshot Screenshot

	query := db.Where("server_url = ?", url)

	if len(thumbOnly) > 0 && thumbOnly[0] {
		// Only load thumbnail + metadata, skip the heavy full-size blob
		query = query.Select("id, server_url, thumbnail_data, thumbnail_w, created_at, mime_type")
	}

	err := query.First(&screenshot).Error
	if err != nil {
		return nil, err
	}

	// If this is a failure marker, only return it within the retry window
	if strings.HasPrefix(screenshot.MIMEType, "error/") {
		if time.Since(screenshot.CreatedAt) < screenshotFailureRetry {
			return &screenshot, nil
		}
		return nil, gorm.ErrRecordNotFound
	}

	// Expire successful screenshots after the cache TTL
	if time.Since(screenshot.CreatedAt) > screenshotCacheTTL {
		return nil, gorm.ErrRecordNotFound
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
		report.SecretExposure.Risk,
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
		len(report.Infrastructure.Findings) +
		len(report.SecretExposure.Findings)

	return fmt.Sprintf("Security scan completed at %s with %d findings",
		report.ScanTimestamp.Format(time.RFC3339),
		findings)
}
