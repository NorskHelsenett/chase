package security

import (
	"context"
	"fmt"
	"math/rand"
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
	StartTime time.Time `gorm:"type:datetime;index" json:"start_time"`
	EndTime   time.Time `gorm:"type:datetime" json:"end_time"`
	Error     string    `gorm:"type:text" json:"error"`
}

// BatchResultStore stores individual server processing results
type BatchResultStore struct {
	ID           string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	BatchJobID   string    `gorm:"type:varchar(50);index" json:"batch_job_id"`
	ServerURL    string    `gorm:"type:varchar(255)" json:"server_url"`
	SecurityScan bool      `gorm:"type:boolean" json:"security_scan_completed"`
	Screenshot   bool      `gorm:"type:boolean" json:"screenshot_completed"`
	Error        string    `gorm:"type:text" json:"error"`
	CreatedAt    time.Time `gorm:"type:datetime;index" json:"created_at"`
}

// BatchResult stores results for individual server processing
type BatchResult struct {
	ServerURL    string `json:"server_url"`
	SecurityScan bool   `json:"security_scan_completed"`
	Screenshot   bool   `json:"screenshot_completed"`
	Error        string `json:"error,omitempty"`
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

	var servers []types.Server
	if err := db.Where("active = ?", true).Find(&servers).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch servers"})
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	jobID := generateJobID()
	job := &BatchJob{
		ID:        jobID,
		Status:    "running",
		Total:     len(servers),
		StartTime: time.Now(),
		Cancel:    cancel,
	}

	jobsMutex.Lock()
	activeJobs[jobID] = job
	jobsMutex.Unlock()

	go processBatch(ctx, servers, job)

	c.JSON(200, gin.H{
		"job_id": jobID,
		"status": "started",
		"total":  len(servers),
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
	if err := updateBatchJob(db, job, nil); err != nil {
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
	})
	screenshotChan := make(chan error)

	// Start security scan in goroutine
	go func() {
		scanner := NewScanner()
		report, err := scanner.ScanWebsite(server.URL)
		securityChan <- struct {
			report *SecurityReport
			err    error
		}{report, err}
	}()

	// Start screenshot capture in goroutine
	go func() {
		// Ensure URL has protocol
		url := server.URL
		if !strings.HasPrefix(url, "http") {
			url = "https://" + url
		}
		err := captureAndSendScreenshot(nil, url)
		screenshotChan <- err
	}()

	// Wait for operations with timeout
	securityTimeout := time.After(30 * time.Second)
	screenshotTimeout := time.After(30 * time.Second)

	// Wait for security scan
	select {
	case <-ctx.Done():
		result.Error = "Operation cancelled"
		return result
	case <-securityTimeout:
		result.Error = "Security scan timed out"
	case securityResult := <-securityChan:
		if securityResult.err != nil {
			result.Error = fmt.Sprintf("Security scan failed: %v", securityResult.err)
		} else {
			result.SecurityScan = true
			// Store the security report
			if err := storeSecurityReport(securityResult.report); err != nil {
				result.Error = fmt.Sprintf("Failed to store security report: %v", err)
			}
		}
	}

	// Wait for screenshot
	select {
	case <-ctx.Done():
		if result.Error == "" {
			result.Error = "Operation cancelled"
		}
		return result
	case <-screenshotTimeout:
		if result.Error == "" {
			result.Error = "Screenshot capture timed out"
		}
	case err := <-screenshotChan:
		if err != nil {
			if result.Error == "" {
				result.Error = fmt.Sprintf("Screenshot capture failed: %v", err)
			}
		} else {
			result.Screenshot = true
		}
	}

	return result
}

func processBatch(ctx context.Context, servers []types.Server, job *BatchJob) {
	db := database.GetDB()

	if err := storeBatchJob(db, job); err != nil {
		job.Error = "Failed to store job: " + err.Error()
		job.Status = "failed"
		return
	}

	results := make(chan BatchResult, len(servers))
	var wg sync.WaitGroup

	workerCount := 3 // Reduced for SQLite
	serverChan := make(chan types.Server, len(servers))

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for server := range serverChan {
				select {
				case <-ctx.Done():
					return
				default:
					result := processServer(ctx, server)
					results <- result
				}
			}
		}()
	}

	for _, server := range servers {
		serverChan <- server
	}
	close(serverChan)

	go func() {
		wg.Wait()
		close(results)
	}()

	updateTicker := time.NewTicker(5 * time.Second)
	defer updateTicker.Stop()

	for {
		select {
		case result, ok := <-results:
			if !ok {
				job.Status = "completed"
				job.EndTime = time.Now()
				if job.Failed == job.Total {
					job.Status = "failed"
				}
				updateBatchJob(db, job, nil)
				return
			}

			jobsMutex.Lock()
			if result.Error != "" {
				job.Failed++
			} else {
				job.Completed++
			}
			jobsMutex.Unlock()

			updateBatchJob(db, job, &result)

		case <-updateTicker.C:
			updateBatchJob(db, job, nil)

		case <-ctx.Done():
			job.Status = "cancelled"
			job.EndTime = time.Now()
			updateBatchJob(db, job, nil)
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
		Error:     job.Error,
	}
	return db.Create(&jobStore).Error
}

func updateBatchJob(db *gorm.DB, job *BatchJob, result *BatchResult) error {
	// Update job status
	if err := db.Model(&BatchJobStore{}).
		Where("id = ?", job.ID).
		Updates(map[string]interface{}{
			"completed": job.Completed,
			"failed":    job.Failed,
			"status":    job.Status,
			"end_time":  job.EndTime,
			"error":     job.Error,
		}).Error; err != nil {
		return err
	}

	// Store result if provided
	if result != nil {
		resultStore := BatchResultStore{
			ID:           generateResultID(),
			BatchJobID:   job.ID,
			ServerURL:    result.ServerURL,
			SecurityScan: result.SecurityScan,
			Screenshot:   result.Screenshot,
			Error:        result.Error,
			CreatedAt:    time.Now(),
		}
		return db.Create(&resultStore).Error
	}

	return nil
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
