package servers

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
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
	db.Create(&result)

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

	// Send push notification for deleted server
	go NotifyServerDeleted(server.URL)

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

	var server Server
	if err := c.ShouldBindJSON(&server); err != nil {
		tx.Rollback()
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var err error
	if server.URL, err = utils.EnsureHTTPS(server.URL); err != nil {
		tx.Rollback()
		c.JSON(400, gin.H{"error": "Invalid URL format"})
		return
	}

	parsedURL, err := url.Parse(server.URL)
	if err != nil {
		tx.Rollback()
		c.JSON(400, gin.H{"error": "Invalid URL format"})
		return
	}

	// Check if host is present
	if parsedURL.Host == "" {
		tx.Rollback()
		c.JSON(400, gin.H{"error": "URL must contain a valid host"})
		return
	}

	// Remove scheme (http/https) from URL
	server.URL = strings.TrimPrefix(strings.TrimPrefix(server.URL, "https://"), "http://")

	// Set default status code if not provided
	if server.ExpectedStatusCode == 0 {
		server.ExpectedStatusCode = 200
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
	go NotifyServerAdded(server.URL)

	// Start the ping in a goroutine without waiting for the result
	go checkServer(server.ID, nil)

	c.JSON(201, server)
}

func GetServer(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")
	var server Server

	// Get the limit from query parameter, default to 30 days in seconds
	limitDays := c.DefaultQuery("limit", "30")
	days, err := strconv.Atoi(limitDays)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid limit parameter"})
		return
	}

	// Calculate the cutoff time
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// Create a subquery to get ping results within the time limit
	subQuery := db.Table("ping_results").
		Where("server_id = ? AND created_at >= ?", id, cutoffTime).
		Order("created_at DESC")

	err = db.Preload("PingResults", func(db *gorm.DB) *gorm.DB {
		return db.Select("*").Table("(?) as ping_results", subQuery)
	}).First(&server, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Server not found"})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, server)
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

	// Send notification if server was manually deactivated
	if wasActive && !server.Active {
		NotifyServerDeactivated(server.URL, "manually deactivated")
	}

	c.JSON(200, server)
}
