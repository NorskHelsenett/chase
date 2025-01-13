package servers

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/security"
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

func GetServersWithSecurityStatus(c *gin.Context) {
	db := database.GetDB()
	var servers []Server

	// Get latest security reports using SQLite compatible syntax
	securityReportSubQuery := db.Select("security_report_records.*").
		Table("security_report_records").
		Where("security_report_records.id IN (?)",
			db.Table("security_report_records").
				Select("MAX(id)").
				Group("server_url"))

	// Get last 10 ping results
	pingSubQuery := db.Select("ping_results.*").
		Table("ping_results").
		Joins("JOIN (SELECT id, ROW_NUMBER() OVER (PARTITION BY server_id ORDER BY created_at DESC) AS rn FROM ping_results) ranked ON ping_results.id = ranked.id").
		Where("ranked.rn <= 10")

	// Load servers with ping results
	err := db.Preload("PingResults", func(db *gorm.DB) *gorm.DB {
		return db.Select("*").Table("(?) as ping_results", pingSubQuery)
	}).Find(&servers).Error

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	type SecuritySummary struct {
		ServerURL  string              `json:"serverUrl"`
		CreatedAt  *time.Time          `json:"createdAt"`
		RiskLevel  *security.RiskLevel `json:"riskLevel"`
		HeaderRisk string              `json:"headerRisk"`
		CertRisk   string              `json:"certRisk"`
		AdminRisk  *security.RiskLevel `json:"adminRisk"`
		APIRisk    *security.RiskLevel `json:"apiRisk"`
	}

	type ServerResponse struct {
		Server
		Security SecuritySummary `json:"security"`
	}

	response := make([]ServerResponse, 0, len(servers))

	// Get all security reports in one query
	var securityReports []security.SecurityReportRecord
	err = db.Table("(?)", securityReportSubQuery).Find(&securityReports).Error
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Create a map for quick lookup of security reports by server URL
	reportsMap := make(map[string]security.SecurityReportRecord)
	for _, report := range securityReports {
		reportsMap[report.ServerURL] = report
	}

	for _, server := range servers {
		// Initialize response with server data
		serverResp := ServerResponse{
			Server: server,
			Security: SecuritySummary{
				ServerURL: server.URL,
			},
		}

		// Look up security report if it exists
		if report, exists := reportsMap[server.URL]; exists {
			var securityReport security.SecurityReport
			if err := json.Unmarshal(report.ReportData, &securityReport); err == nil {
				serverResp.Security.CreatedAt = &report.CreatedAt
				serverResp.Security.RiskLevel = &report.RiskLevel
				serverResp.Security.HeaderRisk = securityReport.Headers.Score
				serverResp.Security.CertRisk = securityReport.Certificate.Grade
				serverResp.Security.AdminRisk = &securityReport.AdminPages.Risk
				serverResp.Security.APIRisk = &securityReport.Swagger.Risk
			}
		}

		response = append(response, serverResp)
	}

	c.JSON(200, response)
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

	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	server.NextCheck = time.Now().Add(time.Duration(server.UpdateInterval) * time.Minute)

	db.Save(&server)
	c.JSON(200, server)
}

func GetServerResults(c *gin.Context) {
	db := database.GetDB()
	var results []PingResult

	// Get query parameters
	id := c.Param("id")
	limitStr := c.DefaultQuery("limit", "10")

	// Parse limit parameter
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10 // default to 10 if invalid input
	}

	// Retrieve results from database
	db.Where("server_id = ?", id).
		Order("created_at desc").
		Limit(limit).
		Find(&results)

	c.JSON(200, results)
}
