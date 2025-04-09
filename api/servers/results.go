package servers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
)

// GetServerResults returns ping results for a specific server
// with optional filtering and pagination
func GetServerResults(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Server ID is required"})
		return
	}

	db := database.GetDB()

	// Check if server exists
	var server Server
	if err := db.First(&server, serverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	// Parse query parameters
	limit := 20 // default limit
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			// Cap the maximum to 100 to prevent excessive queries
			if parsedLimit > 100 {
				limit = 100
			} else {
				limit = parsedLimit
			}
		}
	}

	// Parse time range parameter (in hours)
	timeRange := 24 * 7 // default to 1 week
	if rangeParam := c.Query("range"); rangeParam != "" {
		if parsedRange, err := strconv.Atoi(rangeParam); err == nil && parsedRange > 0 {
			// Cap the maximum to 30 days to prevent excessive queries
			if parsedRange > 24*30 {
				timeRange = 24 * 30
			} else {
				timeRange = parsedRange
			}
		}
	}

	// Calculate time threshold
	timeThreshold := time.Now().Add(-time.Duration(timeRange) * time.Hour)

	// Query ping results for this server
	var results []PingResult
	if err := db.Where("server_id = ? AND timestamp > ?", serverID, timeThreshold).
		Order("timestamp DESC").
		Limit(limit).
		Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ping results"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetServersWithSecurityStatus fetches all servers with their latest ping result but without detailed ping history
// This makes the initial load much faster
func GetServersWithSecurityStatus(c *gin.Context) {
	db := database.GetDB()

	// Get active filter from query parameter
	activeFilter := c.Query("active")

	// Base query for servers
	query := db.Model(&Server{})

	// Apply active filter if specified
	if activeFilter != "" {
		isActive := activeFilter == "true"
		query = query.Where("active = ?", isActive)
	}

	// Get all servers
	var servers []Server
	if err := query.Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch servers"})
		return
	}

	// For each server, get only the latest ping result
	for i := range servers {
		var latestPing PingResult
		if err := db.Where("server_id = ?", servers[i].ID).
			Order("timestamp DESC").
			Limit(1).
			First(&latestPing).Error; err == nil {
			servers[i].PingResults = []PingResult{latestPing}
		} else {
			// If no ping results, initialize empty array
			servers[i].PingResults = []PingResult{}
		}
	}

	c.JSON(http.StatusOK, servers)
}
