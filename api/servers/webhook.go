package servers

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/security"
)

type webhookServerRequest struct {
	Domain      string    `json:"domain" binding:"required"`
	Timestamp   time.Time `json:"timestamp"`
	CertType    string    `json:"cert_type"`
	CommonName  string    `json:"common_name"`
	Issuer      string    `json:"issuer"`
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	AllDomains  []string  `json:"all_domains"`
	MatchedWith string    `json:"matched_with"`
}

func AddServerFromWebhook(c *gin.Context) {
	db := database.GetDB()

	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(500, gin.H{"error": "Failed to start transaction"})
		return
	}

	var request webhookServerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		tx.Rollback()
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	server := defaultServerModel()
	server.URL = request.Domain
	server.Comment = strings.TrimSpace(c.GetHeader("User-Agent"))

	var err error
	if server.URL, err = normalizeServerURL(server.URL); err != nil {
		tx.Rollback()
		c.JSON(400, gin.H{"error": "Invalid URL format"})
		return
	}

	server.NextCheck = time.Now().Add(time.Duration(server.UpdateInterval) * time.Minute)

	if err := tx.Create(&server).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(409, gin.H{"error": "Server URL already exists"})
			return
		}
		c.JSON(500, gin.H{"error": "Failed to create server"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to commit transaction"})
		return
	}

	BroadcastServerAdded(server)
	go NotifyServerAdded(server.ID, server.URL)
	go checkServer(server.ID, nil)
	go security.RunBackgroundScan(server.URL)

	// Capture screenshot for the new server
	go func() {
		if err := security.CaptureScreenshotForDomain(server.URL); err != nil {
			log.Printf("Screenshot capture failed for webhook server %s: %v", server.URL, err)
		}
	}()

	c.JSON(201, server)
}
