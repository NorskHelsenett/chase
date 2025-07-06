// BatchImportServers handles importing multiple servers at once
package servers

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/utils"
)

// BatchImportRequest represents the data sent in a batch import request
type BatchImportRequest struct {
	Sites    []string `json:"sites" binding:"required"` // URLs to import (already processed for separators on client side)
	Settings struct {
		UpdateInterval int  `json:"update_interval"`
		FollowRedirect bool `json:"follow_redirect"`
		AllowInsecure  bool `json:"allow_insecure"`
	} `json:"settings"`
}

// BatchImportResponse represents the response for a batch import operation
type BatchImportResponse struct {
	Total    int      `json:"total"`
	Imported int      `json:"imported"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

// BatchImportServers handles importing multiple servers at once
func BatchImportServers(c *gin.Context) {
	var request BatchImportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	
	// Start a database transaction
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(500, gin.H{"error": "Failed to start transaction"})
		return
	}

	response := BatchImportResponse{
		Total: len(request.Sites),
	}

	// Default settings
	if request.Settings.UpdateInterval <= 0 {
		request.Settings.UpdateInterval = 15 // Default to 15 minutes
	}

	// Process each URL
	for _, site := range request.Sites {
		server := Server{
			URL:               site,
			Active:            true,
			FollowRedirect:    request.Settings.FollowRedirect,
			AllowInsecure:     request.Settings.AllowInsecure,
			ExpectedStatusCode: 200,
			UpdateInterval:    request.Settings.UpdateInterval,
		}

		var err error
		if server.URL, err = utils.EnsureHTTPS(server.URL); err != nil {
			response.Failed++
			response.Errors = append(response.Errors, "Invalid URL format: "+site)
			continue
		}

		parsedURL, err := url.Parse(server.URL)
		if err != nil {
			response.Failed++
			response.Errors = append(response.Errors, "Invalid URL format: "+site)
			continue
		}

		// Check if host is present
		if parsedURL.Host == "" {
			response.Failed++
			response.Errors = append(response.Errors, "URL must contain a valid host: "+site)
			continue
		}

		// Remove scheme (http/https) from URL for storage
		server.URL = strings.TrimPrefix(strings.TrimPrefix(server.URL, "https://"), "http://")

		server.NextCheck = time.Now().Add(time.Duration(server.UpdateInterval) * time.Minute)

		// Attempt to create the server in the transaction
		if err := tx.Create(&server).Error; err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "Duplicate entry") {
				response.Failed++
				response.Errors = append(response.Errors, "Server URL already exists: "+site)
				continue
			}
			response.Failed++
			response.Errors = append(response.Errors, "Failed to create server: "+site)
			continue
		}

		// Queue the server for checking
		go checkServer(server.ID, nil)
		
		response.Imported++
	}

	// Commit the transaction if we imported at least one server
	if response.Imported > 0 {
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to commit transaction"})
			return
		}
	} else {
		// If nothing was imported, rollback
		tx.Rollback()
	}

	c.JSON(200, response)
}
