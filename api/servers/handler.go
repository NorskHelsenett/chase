package servers

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"gorm.io/gorm"
)

// Force check endpoint
func ForceCheckServer(c *gin.Context) {
	db := database.GetDB()
	var server Server
	if err := db.First(&server, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "Server not found"})
		return
	}

	result := pingServer(server)
	db.Create(&result)

	// Reset the failure count if successful
	if result.Error == "" && result.StatusCode < 500 {
		db.Model(&server).Updates(map[string]interface{}{
			"failure_count": 0,
			"last_success":  time.Now(),
			"next_check":    time.Now().Add(time.Hour),
		})
	}

	c.JSON(200, result)
}

// API Handlers (remaining handlers stay the same)
func AddServer(c *gin.Context) {
	db := database.GetDB()
	var server Server
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if server.ExpectedStatusCode == 0 {
		server.ExpectedStatusCode = 200
	}

	server.NextCheck = time.Now() // Set initial check time
	db.Create(&server)
	c.JSON(201, server)
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
