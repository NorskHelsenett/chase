package servers

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/security"
	"github.com/norskhelsenett/chase/types"
	"gorm.io/gorm"
)

func checkServer(serverID uint, resultChan chan<- any) {
	db := database.GetDB()
	var server Server
	if err := db.First(&server, serverID).Error; err != nil {
		if resultChan != nil {
			resultChan <- nil
		}
		return
	}

	result := pingServer(server)

	// Update site metadata if extracted during ping
	if result.siteMetadata.Favicon != "" {
		server.Favicon = result.siteMetadata.Favicon
	}
	if result.siteMetadata.Title != "" {
		server.SiteTitle = result.siteMetadata.Title
	}
	if result.siteMetadata.Description != "" {
		server.SiteDescription = result.siteMetadata.Description
	}
	if result.siteMetadata.OGImage != "" {
		server.OGImage = result.siteMetadata.OGImage
	}
	db.Save(&server)

	db.Create(&result)

	// Broadcast to SSE clients
	BroadcastPing(serverID, server.ExpectedStatusCode, result)

	if resultChan != nil {
		resultChan <- result
	}
}

func ForceCheckServer(c *gin.Context) {
	db := database.GetDB()
	var server Server
	if err := db.First(&server, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "Server not found"})
		return
	}

	resultChan := make(chan any)
	go checkServer(server.ID, resultChan)
	result := <-resultChan

	c.JSON(200, result)
}

func DeleteServer(c *gin.Context) {
	db := database.GetDB()

	// Get server ID from URL parameter
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	// Start a database transaction
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(500, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Check if server exists
	var server Server
	if err := tx.First(&server, serverID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Server not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Failed to query server"})
		return
	}

	// Soft delete the server and its related ping results
	// First delete related ping results
	// To hard delete to tx.Unscoped()
	if err := tx.Where("server_id = ?", serverID).Delete(&PingResult{}).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to delete server ping results"})
		return
	}

	// Then delete the server itself
	// To hard delete to tx.Unscoped()
	if err := tx.Delete(&server).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to delete server"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Invalidate geo cache since server was deleted
	go InvalidateGeoResponseCache()

	// Send push notification for deleted server (after committing, but before losing server data)
	go NotifyServerDeleted(server.ID, server.URL)

	c.Status(204)
}

func AddServer(c *gin.Context) {
	db := database.GetDB()

	// Start a database transaction
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(500, gin.H{"error": "Failed to start transaction"})
		return
	}

	var request serverCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		tx.Rollback()
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	server := serverFromCreateRequest(request)
	var err error

	if server.URL, err = normalizeServerURL(server.URL); err != nil {
		tx.Rollback()
		c.JSON(400, gin.H{"error": "Invalid URL format"})
		return
	}

	server.NextCheck = time.Now().Add(time.Duration(server.UpdateInterval) * time.Minute)

	// Attempt to create the server in the transaction
	if err := tx.Create(&server).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(409, gin.H{"error": "Server URL already exists"})
			return
		}
		c.JSON(500, gin.H{"error": "Failed to create server"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Send push notification for new server
	go NotifyServerAdded(server.ID, server.URL)

	// Invalidate geo cache since we have a new server
	go InvalidateGeoResponseCache()

	// Start ping and security scan in background
	go checkServer(server.ID, nil)
	go security.RunBackgroundScan(server.URL)

	// Capture screenshot for the new server
	go func() {
		if err := security.CaptureScreenshotForDomain(server.URL); err != nil {
			log.Printf("Screenshot capture failed for new server %s: %v", server.URL, err)
		}
	}()

	c.JSON(201, server)
}

type serverDetailResponse struct {
	ID                  uint          `json:"ID"`
	CreatedAt           time.Time     `json:"CreatedAt"`
	UpdatedAt           time.Time     `json:"UpdatedAt"`
	DeletedAt           *time.Time    `json:"DeletedAt"`
	URL                 string        `json:"url"`
	Active              bool          `json:"active"`
	FollowRedirect      bool          `json:"follow_redirect"`
	NextCheck           time.Time     `json:"next_check"`
	AllowInsecure       bool          `json:"allow_insecure"`
	ExpectedStatusCode  int           `json:"expected_status"`
	Comment             string        `json:"comment"`
	UpdateInterval      int           `json:"update_interval"`
	PingResults         []pingSummary `json:"ping_results"`
	SecurityRiskLevel   string        `json:"security_risk_level,omitempty"`
	SecurityDescription string        `json:"security_description,omitempty"`
	SecurityScanTime    time.Time     `json:"security_scan_time,omitempty"`
	HeaderScore         string        `json:"header_score,omitempty"`
	CertScore           string        `json:"cert_score,omitempty"`
	AdminRisk           string        `json:"admin_risk,omitempty"`
	APIRisk             string        `json:"api_risk,omitempty"`
}

func GetServer(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")
	var server Server

	limitDays := c.DefaultQuery("limit", "30")
	days, err := strconv.Atoi(limitDays)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid limit parameter"})
		return
	}

	const defaultMaxResults = 2000
	const absoluteMaxResults = 5000

	maxResults := defaultMaxResults
	if maxParam := c.Query("max"); maxParam != "" {
		if parsedMax, convErr := strconv.Atoi(maxParam); convErr == nil && parsedMax > 0 {
			if parsedMax > absoluteMaxResults {
				maxResults = absoluteMaxResults
			} else {
				maxResults = parsedMax
			}
		}
	}

	includeDetail := strings.ToLower(c.Query("includeDetail")) == "true"

	cutoffTime := time.Now().AddDate(0, 0, -days)

	if err := db.First(&server, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Server not found"})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var pingResults []PingResult
	pingQuery := db.Model(&PingResult{}).
		Where("server_id = ? AND timestamp >= ?", id, cutoffTime).
		Order("timestamp DESC").
		Limit(maxResults)

	if includeDetail {
		pingQuery = pingQuery.Preload("PingDetail")
	} else {
		pingQuery = pingQuery.Select("status_code, response_time, error, timestamp")
	}

	if err := pingQuery.Find(&pingResults).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	pingSummaries := make([]pingSummary, len(pingResults))
	for i, pr := range pingResults {
		status := pr.StatusCode
		if pr.Error != "" {
			status = 0
		}
		pingSummaries[i] = pingSummary{
			StatusCode:   status,
			ResponseTime: pr.ResponseTime,
			Error:        pr.Error,
			Timestamp:    pr.Timestamp,
		}
		if includeDetail {
			pingSummaries[i].Detail = pr.PingDetail
		}
	}

	var deletedAt *time.Time
	if server.DeletedAt.Valid {
		t := server.DeletedAt.Time
		deletedAt = &t
	}

	response := serverDetailResponse{
		ID:                  server.ID,
		CreatedAt:           server.CreatedAt,
		UpdatedAt:           server.UpdatedAt,
		DeletedAt:           deletedAt,
		URL:                 server.URL,
		Active:              server.Active,
		FollowRedirect:      server.FollowRedirect,
		NextCheck:           server.NextCheck,
		AllowInsecure:       server.AllowInsecure,
		ExpectedStatusCode:  server.ExpectedStatusCode,
		Comment:             server.Comment,
		UpdateInterval:      server.UpdateInterval,
		PingResults:         pingSummaries,
		SecurityRiskLevel:   server.SecurityRiskLevel,
		SecurityDescription: server.SecurityDescription,
		SecurityScanTime:    server.SecurityScanTime,
		HeaderScore:         server.HeaderScore,
		CertScore:           server.CertScore,
		AdminRisk:           server.AdminRisk,
		APIRisk:             server.APIRisk,
	}

	c.JSON(200, response)
}

func GetServers(c *gin.Context) {
	db := database.GetDB()
	var servers []Server

	// Create a subquery to get the last 10 results for each server
	subQuery := db.Select("ping_results.*").
		Table("ping_results").
		Joins("JOIN (SELECT id, ROW_NUMBER() OVER (PARTITION BY server_id ORDER BY created_at DESC) AS rn FROM ping_results) ranked ON ping_results.id = ranked.id").
		Where("ranked.rn <= 10")

	err := db.Preload("PingResults", func(db *gorm.DB) *gorm.DB {
		return db.Select("*").Table("(?) as ping_results", subQuery)
	}).Find(&servers).Error

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, servers)
}

func PatchServer(c *gin.Context) {
	db := database.GetDB()
	var server Server
	if err := db.First(&server, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "Server not found"})
		return
	}

	var patch types.ServerPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Only update fields that are present in the request
	if patch.URL != nil {
		server.URL = *patch.URL
	}
	if patch.Active != nil {
		server.Active = *patch.Active
	}
	if patch.FollowRedirect != nil {
		server.FollowRedirect = *patch.FollowRedirect
	}
	if patch.AllowInsecure != nil {
		server.AllowInsecure = *patch.AllowInsecure
	}
	if patch.ExpectedStatusCode != nil {
		server.ExpectedStatusCode = *patch.ExpectedStatusCode
	}
	if patch.Comment != nil {
		server.Comment = *patch.Comment
	}
	if patch.UpdateInterval != nil {
		server.UpdateInterval = *patch.UpdateInterval
		// Update NextCheck when interval changes
		server.NextCheck = time.Now().Add(time.Duration(server.UpdateInterval) * time.Minute)
	}

	if err := db.Save(&server).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to update server"})
		return
	}

	// Invalidate geo cache since server properties changed
	go InvalidateGeoResponseCache()

	c.JSON(200, server)
}

func UpdateServer(c *gin.Context) {
	db := database.GetDB()
	var server Server
	if err := db.First(&server, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "Server not found"})
		return
	}

	// Store the original active state before updating
	wasActive := server.Active

	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	server.NextCheck = time.Now().Add(time.Duration(server.UpdateInterval) * time.Minute)

	db.Save(&server)

	// Invalidate geo cache since server was updated
	go InvalidateGeoResponseCache()

	// Send notification if server was manually deactivated
	if wasActive && !server.Active {
		NotifyServerDeactivated(server.ID, server.URL, "manually deactivated")
	}

	c.JSON(200, server)
}
