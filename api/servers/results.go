package servers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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
	if err := query.Debug().Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch servers"})
		return
	}

	// For each server, get only the latest 20 ping result
	for i := range servers {
		var latestPings []PingResult
		if err := db.Where("server_id = ?", servers[i].ID).
			Order("timestamp DESC").
			Limit(20).
			Find(&latestPings).Error; err == nil {
			servers[i].PingResults = latestPings
		} else {
			// If no ping results, initialize empty array
			servers[i].PingResults = []PingResult{}
		}
	}

	// add security report status, but only include the score and/or risk
	for i := range servers {
		var securityReport struct {
			ID          uint      `json:"id"`
			RiskLevel   string    `json:"risk_level"`
			Description string    `json:"description"`
			CreatedAt   time.Time `json:"created_at"`
			ReportData  []byte    `json:"report_data"`
		}

		// Query the security report record for the server's URL
		serverURL := strings.TrimPrefix(strings.TrimPrefix(servers[i].URL, "https://"), "http://")
		if err := db.Table("security_report_records").
			Where("server_url = ?", serverURL).
			Order("created_at DESC").
			First(&securityReport).Error; err == nil {
			// Add security report metadata to server
			servers[i].SecurityRiskLevel = securityReport.RiskLevel
			servers[i].SecurityDescription = securityReport.Description
			servers[i].SecurityScanTime = securityReport.CreatedAt

			// Extract additional security details from report data if available
			if len(securityReport.ReportData) > 0 {
				var fullReport struct {
					Headers struct {
						Score string `json:"score"`
						Risk  string `json:"risk"`
					} `json:"headers"`
					Certificate struct {
						Grade string `json:"grade"`
						Risk  string `json:"risk"`
					} `json:"certificate"`
					AdminPages struct {
						Risk string `json:"risk"`
					} `json:"adminPages"`
					Swagger struct {
						Risk string `json:"risk"`
					} `json:"swagger"`
				}

				if err := json.Unmarshal(securityReport.ReportData, &fullReport); err == nil {
					// Populate additional security details
					servers[i].HeaderScore = fullReport.Headers.Score
					servers[i].CertScore = fullReport.Certificate.Grade
					servers[i].AdminRisk = string(fullReport.AdminPages.Risk)
					servers[i].APIRisk = string(fullReport.Swagger.Risk)
				}
			}
		}
	}

	c.JSON(http.StatusOK, servers)
}
