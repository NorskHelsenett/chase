package security

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/types"
	"gorm.io/gorm"
)

// BatchJobStore represents the persistent storage for batch jobs
type BatchJobStore struct {
	ID        string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Status    string    `gorm:"type:varchar(20);index" json:"status"`
	Total     int       `gorm:"type:integer" json:"total"`
	Completed int       `gorm:"type:integer" json:"completed"`
	Failed    int       `gorm:"type:integer" json:"failed"`
	StartTime time.Time `gorm:"index" json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Errors    []string  `gorm:"-" json:"errors"`
}

// BatchResultStore stores individual server processing results
type BatchResultStore struct {
	ID              string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	BatchJobID      string    `gorm:"type:varchar(50);index" json:"batch_job_id"`
	ServerURL       string    `gorm:"type:varchar(255)" json:"server_url"`
	SecurityScan    bool      `gorm:"type:boolean" json:"security_scan_completed"`
	Screenshot      bool      `gorm:"type:boolean" json:"screenshot_completed"`
	Error           string    `gorm:"type:text" json:"error"`
	SecurityError   string    `gorm:"type:text" json:"security_error"`
	ScreenshotError string    `gorm:"type:text" json:"screenshot_error"`
	CreatedAt       time.Time `gorm:"index" json:"created_at"`
}

// BatchResult stores results for individual server processing
type BatchResult struct {
	ServerURL       string `json:"server_url"`
	SecurityScan    bool   `json:"security_scan_completed"`
	Screenshot      bool   `json:"screenshot_completed"`
	Error           string `json:"error,omitempty"`
	SecurityError   string `json:"security_error,omitempty"`
	ScreenshotError string `json:"screenshot_error,omitempty"`
}

// BatchJob represents a running batch operation
type BatchJob struct {
	ID        string             `json:"id"`
	Status    string             `json:"status"`
	Total     int                `json:"total"`
	Completed int                `json:"completed"`
	Failed    int                `json:"failed"`
	StartTime time.Time          `json:"start_time"`
	EndTime   time.Time          `json:"end_time,omitempty"`
	Cancel    context.CancelFunc `json:"-"`
	Error     string             `json:"error,omitempty"`
	Results   []BatchResult      `json:"results,omitempty"`
}

// Global job tracker
var (
	activeJobs = make(map[string]*BatchJob)
	jobsMutex  sync.RWMutex
)

// ListBatchesHandler returns a list of all batch jobs
func ListBatchesHandler(c *gin.Context) {
	db := database.GetDB()

	status := c.Query("status")

	// Convert limit and offset to integers with default values
	limitInt := 50
	offsetInt := 0

	if limit := c.Query("limit"); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 {
			limitInt = parsed
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if parsed, err := strconv.Atoi(offset); err == nil && parsed >= 0 {
			offsetInt = parsed
		}
	}

	query := db.Model(&BatchJobStore{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to count batch jobs"})
		return
	}

	var batchJobs []BatchJobStore
	if err := query.Order("start_time DESC").
		Limit(limitInt).
		Offset(offsetInt).
		Find(&batchJobs).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch batch jobs"})
		return
	}

	// Get results for batch jobs
	for i := range batchJobs {
		var results []BatchResultStore
		if err := db.Where("batch_job_id = ?", batchJobs[i].ID).
			Find(&results).Error; err != nil {
			continue
		}

		// Initialize errors slice
		batchJobs[i].Errors = make([]string, 0)

		// Collect non-empty errors from results
		for _, result := range results {
			if result.Error != "" {
				errorWithDomain := fmt.Sprintf("%s: %s", result.ServerURL, result.Error)
				batchJobs[i].Errors = append(batchJobs[i].Errors, errorWithDomain)
			}
		}
	}

	// Combine with active jobs
	jobsMutex.RLock()
	activeJobsList := make([]BatchJob, 0, len(activeJobs))
	for _, job := range activeJobs {
		if status == "" || job.Status == status {
			jobCopy := *job
			jobCopy.Cancel = nil
			activeJobsList = append(activeJobsList, jobCopy)
		}
	}
	jobsMutex.RUnlock()

	c.JSON(200, gin.H{
		"total":          total,
		"active_jobs":    activeJobsList,
		"completed_jobs": batchJobs,
		"pagination": gin.H{
			"limit":  limitInt,
			"offset": offsetInt,
		},
	})
}

// StartBatchHandler initiates a new batch job
func StartBatchHandler(c *gin.Context) {
	db := database.GetDB()

	// Using a subquery to get the latest ping result for each server
	// Then LEFT JOIN with servers table to get all active servers
	query := `
		WITH LatestPings AS (
			SELECT server_id, status_code, error,
				   ROW_NUMBER() OVER (PARTITION BY server_id ORDER BY timestamp DESC) as rn
			FROM ping_results
		)
		SELECT s.*, 
			   lp.status_code as last_status_code,
			   lp.error as last_error
		FROM servers s
		LEFT JOIN LatestPings lp ON s.id = lp.server_id AND lp.rn = 1
		WHERE s.active = true`

	type ServerWithPing struct {
		types.Server
		LastStatusCode *int    `gorm:"column:last_status_code"`
		LastError      *string `gorm:"column:last_error"`
	}

	var serversWithPing []ServerWithPing
	if err := db.Raw(query).Scan(&serversWithPing).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch servers with ping results"})
		return
	}

	var eligibleServers []types.Server
	for _, s := range serversWithPing {
		// Server is eligible if:
		// 1. No ping results yet (LastStatusCode is nil) OR
		// 2. Last ping was successful (no error and status 200-399)
		if s.LastStatusCode == nil ||
			(s.LastError == nil || *s.LastError == "") &&
				*s.LastStatusCode >= 200 && *s.LastStatusCode < 400 {
			eligibleServers = append(eligibleServers, s.Server)
		}
	}

	if len(eligibleServers) == 0 {
		c.JSON(400, gin.H{
			"error": "No eligible servers found - all servers have errors in their last ping",
		})
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	jobID := generateJobID()
	job := &BatchJob{
		ID:        jobID,
		Status:    "running",
		Total:     len(eligibleServers),
		StartTime: time.Now(),
		Cancel:    cancel,
	}

	jobsMutex.Lock()
	activeJobs[jobID] = job
	jobsMutex.Unlock()

	go processBatch(ctx, eligibleServers, job)

	c.JSON(200, gin.H{
		"job_id": jobID,
		"status": "started",
		"total":  len(eligibleServers),
	})
}

// CancelBatchHandler cancels a running batch job
func CancelBatchHandler(c *gin.Context) {
	jobID := c.Param("jobID")

	jobsMutex.Lock()
	defer jobsMutex.Unlock()

	job, exists := activeJobs[jobID]
	if !exists {
		c.JSON(404, gin.H{"error": "Job not found"})
		return
	}

	if job.Status != "running" {
		c.JSON(400, gin.H{"error": "Job is not running"})
		return
	}

	job.Cancel()
	job.Status = "cancelled"
	job.EndTime = time.Now()

	// Update job status in database
	db := database.GetDB()
	if err := updateBatchJob(db, job); err != nil {
		c.JSON(500, gin.H{"error": "Failed to update job status"})
		return
	}

	c.JSON(200, gin.H{"status": "cancelled"})
}

// GetBatchStatusHandler returns the current status of a batch job
func GetBatchStatusHandler(c *gin.Context) {
	jobID := c.Param("jobID")
	db := database.GetDB()

	// First check active jobs
	jobsMutex.RLock()
	job, exists := activeJobs[jobID]
	jobsMutex.RUnlock()

	if exists {
		jobCopy := *job
		jobCopy.Cancel = nil // Don't expose cancel func in JSON
		c.JSON(200, jobCopy)
		return
	}

	// If not active, check database
	var storedJob BatchJobStore
	if err := db.Where("id = ?", jobID).First(&storedJob).Error; err != nil {
		c.JSON(404, gin.H{"error": "Job not found"})
		return
	}

	// Get results for this job
	var results []BatchResultStore
	if err := db.Where("batch_job_id = ?", jobID).Find(&results).Error; err == nil {
		// Include results in response if found
	}

	c.JSON(200, storedJob)
}

func processServer(ctx context.Context, server types.Server) BatchResult {
	result := BatchResult{
		ServerURL: server.URL,
	}

	// Create channels for each operation
	securityChan := make(chan struct {
		report *SecurityReport
		err    error
	}, 1)
	screenshotChan := make(chan error, 1)

	// Start security scan in goroutine
	securityCtx, cancelSecurity := context.WithTimeout(ctx, 30*time.Second)
	go func() {
		defer cancelSecurity()
		scanner := NewScanner(0)
		report, err := scanner.ScanWebsite(securityCtx, server.URL)
		select {
		case securityChan <- struct {
			report *SecurityReport
			err    error
		}{report, err}:
		case <-securityCtx.Done():
		}
	}()

	// Start screenshot capture in goroutine
	screenshotCtx, cancelScreenshot := context.WithTimeout(ctx, 30*time.Second)
	go func() {
		defer cancelScreenshot()
		// Ensure URL has protocol
		url := server.URL
		if !strings.HasPrefix(url, "http") {
			url = "https://" + url
		}
		// Bulk path: one attempt, fail fast. A slow site shouldn't hold a worker
		// through the backoff — the next job run retries it.
		err := captureAndSendScreenshot(nil, url, false, 0, 1)
		select {
		case screenshotChan <- err:
		case <-screenshotCtx.Done():
		}
	}()

	// Wait for security scan
	select {
	case <-ctx.Done():
		result.SecurityError = "Operation cancelled"
		return result
	case <-securityCtx.Done():
		result.SecurityError = "Security scan timed out"
	case securityResult := <-securityChan:
		if securityResult.err != nil {
			result.SecurityError = fmt.Sprintf("Security scan failed: %v", securityResult.err)
		} else {
			result.SecurityScan = true
			// Store the security report
			if err := storeSecurityReport(securityResult.report); err != nil {
				result.SecurityError = fmt.Sprintf("Failed to store security report: %v", err)
			}
		}
	}

	// Wait for screenshot
	select {
	case <-ctx.Done():
		if result.ScreenshotError == "" {
			result.ScreenshotError = "Operation cancelled"
		}
		return result
	case <-screenshotCtx.Done():
		if result.ScreenshotError == "" {
			result.ScreenshotError = "Screenshot capture timed out"
		}
	case err := <-screenshotChan:
		if err != nil {
			if result.ScreenshotError == "" {
				result.ScreenshotError = fmt.Sprintf("Screenshot capture failed: %v", err)
			}
		} else {
			result.Screenshot = true
		}
	}

	switch {
	case result.SecurityError != "" && result.ScreenshotError != "":
		result.Error = fmt.Sprintf("security: %s; screenshot: %s", result.SecurityError, result.ScreenshotError)
	case result.SecurityError != "":
		result.Error = result.SecurityError
	case result.ScreenshotError != "":
		result.Error = result.ScreenshotError
	}

	return result
}

func processBatch(ctx context.Context, servers []types.Server, job *BatchJob) {
	db := database.GetDB()

	if err := storeBatchJob(db, job); err != nil {
		job.Error = "Failed to store job: " + err.Error()
		job.Status = "failed"
		job.EndTime = time.Now()

		jobsMutex.Lock()
		delete(activeJobs, job.ID)
		jobsMutex.Unlock()

		return
	}

	results := make(chan BatchResult)
	var wg sync.WaitGroup

	workerCount := getBatchWorkerCount()
	serverChan := make(chan types.Server, workerCount*2)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for server := range serverChan {
				select {
				case <-ctx.Done():
					return
				case results <- processServer(ctx, server):
				}
			}
		}()
	}

	go func() {
		defer close(serverChan)
		for _, server := range servers {
			select {
			case <-ctx.Done():
				return
			case serverChan <- server:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	updateTicker := time.NewTicker(5 * time.Second)
	defer updateTicker.Stop()

	cleanupJob := func() {
		jobsMutex.Lock()
		delete(activeJobs, job.ID)
		jobsMutex.Unlock()
	}

	pendingResults := make([]BatchResult, 0, 10)
	flushResults := func() {
		if len(pendingResults) == 0 {
			return
		}
		if err := storeBatchResults(db, job.ID, pendingResults); err != nil {
			log.Printf("failed to store batch results: %v", err)
		}
		pendingResults = pendingResults[:0]
	}

	for {
		select {
		case result, ok := <-results:
			if !ok {
				flushResults()
				job.Status = "completed"
				job.EndTime = time.Now()
				if job.Failed == job.Total {
					job.Status = "failed"
				}
				updateBatchJob(db, job)
				cleanupJob()
				return
			}

			jobsMutex.Lock()
			if result.Error != "" {
				job.Failed++
			} else {
				job.Completed++
			}
			jobsMutex.Unlock()

			pendingResults = append(pendingResults, result)
			if len(pendingResults) >= 10 {
				flushResults()
			}

		case <-updateTicker.C:
			flushResults()
			updateBatchJob(db, job)

		case <-ctx.Done():
			flushResults()
			job.Status = "cancelled"
			job.EndTime = time.Now()
			updateBatchJob(db, job)
			cleanupJob()
			return
		}
	}
}

// Helper functions for database operations
func storeBatchJob(db *gorm.DB, job *BatchJob) error {
	jobStore := BatchJobStore{
		ID:        job.ID,
		Status:    job.Status,
		Total:     job.Total,
		Completed: job.Completed,
		Failed:    job.Failed,
		StartTime: job.StartTime,
		EndTime:   job.EndTime,
	}
	return db.Create(&jobStore).Error
}

func updateBatchJob(db *gorm.DB, job *BatchJob) error {
	return db.Model(&BatchJobStore{}).
		Where("id = ?", job.ID).
		Updates(map[string]interface{}{
			"completed": job.Completed,
			"failed":    job.Failed,
			"status":    job.Status,
			"end_time":  job.EndTime,
		}).Error
}

func storeBatchResults(db *gorm.DB, jobID string, results []BatchResult) error {
	if len(results) == 0 {
		return nil
	}

	stores := make([]BatchResultStore, 0, len(results))
	now := time.Now()
	for _, result := range results {
		stores = append(stores, BatchResultStore{
			ID:              generateResultID(),
			BatchJobID:      jobID,
			ServerURL:       result.ServerURL,
			SecurityScan:    result.SecurityScan,
			Screenshot:      result.Screenshot,
			Error:           result.Error,
			SecurityError:   result.SecurityError,
			ScreenshotError: result.ScreenshotError,
			CreatedAt:       now,
		})
	}
	return db.Create(&stores).Error
}

func generateJobID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func generateResultID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// RunBatchSecurityScan runs a batch security scan on all eligible active servers.
// Designed to be called from the scheduler.
func RunBatchSecurityScan(ctx context.Context, progress func(string)) (string, error) {
	db := database.GetDB()

	query := `
		WITH LatestPings AS (
			SELECT server_id, status_code, error,
				   ROW_NUMBER() OVER (PARTITION BY server_id ORDER BY timestamp DESC) as rn
			FROM ping_results
		)
		SELECT s.*,
			   lp.status_code as last_status_code,
			   lp.error as last_error
		FROM servers s
		LEFT JOIN LatestPings lp ON s.id = lp.server_id AND lp.rn = 1
		WHERE s.active = true`

	type ServerWithPing struct {
		types.Server
		LastStatusCode *int    `gorm:"column:last_status_code"`
		LastError      *string `gorm:"column:last_error"`
	}

	var serversWithPing []ServerWithPing
	if err := db.Raw(query).Scan(&serversWithPing).Error; err != nil {
		return "", fmt.Errorf("failed to fetch servers: %w", err)
	}

	var eligibleServers []types.Server
	for _, s := range serversWithPing {
		if s.LastStatusCode == nil ||
			(s.LastError == nil || *s.LastError == "") &&
				*s.LastStatusCode >= 200 && *s.LastStatusCode < 400 {
			eligibleServers = append(eligibleServers, s.Server)
		}
	}

	if len(eligibleServers) == 0 {
		return "no eligible servers found", nil
	}

	progress(fmt.Sprintf("scanning %d servers", len(eligibleServers)))

	jobID := generateJobID()
	job := &BatchJob{
		ID:        jobID,
		Status:    "running",
		Total:     len(eligibleServers),
		StartTime: time.Now(),
	}

	// Use the existing batch infrastructure but hook into progress
	batchCtx, cancel := context.WithCancel(ctx)
	job.Cancel = cancel

	jobsMutex.Lock()
	activeJobs[jobID] = job
	jobsMutex.Unlock()

	// Run synchronously (we're already in a goroutine from the scheduler)
	processBatch(batchCtx, eligibleServers, job)

	jobsMutex.RLock()
	completed := job.Completed
	failed := job.Failed
	total := job.Total
	status := job.Status
	jobsMutex.RUnlock()

	if status == "cancelled" {
		return fmt.Sprintf("cancelled — %d/%d completed, %d failed", completed, total, failed), ctx.Err()
	}

	return fmt.Sprintf("scanned %d servers — %d ok, %d failed", total, completed, failed), nil
}

func getBatchWorkerCount() int {
	value := os.Getenv("BATCH_WORKERS")
	if value == "" {
		return 3
	}

	if count, err := strconv.Atoi(value); err == nil && count > 0 {
		return count
	}

	return 3
}
