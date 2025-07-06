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
	Sites          []string `json:"sites" binding:"required"` // URLs to import (already processed for separators on client side)
	UpdateExisting bool     `json:"update_existing"`          // Whether to update existing servers or skip them
	Settings       struct {
		UpdateInterval int  `json:"update_interval"`
		FollowRedirect bool `json:"follow_redirect"`
		AllowInsecure  bool `json:"allow_insecure"`
		Active         bool `json:"active"`
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
		var err error
		var parsedURL *url.URL
		var formattedURL string

		// Process URL format
		if formattedURL, err = utils.EnsureHTTPS(site); err != nil {
			response.Failed++
			response.Errors = append(response.Errors, "Invalid URL format: "+site)
			continue
		}

		parsedURL, err = url.Parse(formattedURL)
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
		cleanURL := strings.TrimPrefix(strings.TrimPrefix(formattedURL, "https://"), "http://")

		// Check if server already exists
		var existingServer Server
		if tx.Where("url = ?", cleanURL).First(&existingServer).RowsAffected > 0 {
			// Server already exists
			if request.UpdateExisting {
				// Update existing server with new settings
				updates := map[string]interface{}{
					"follow_redirect": request.Settings.FollowRedirect,
					"allow_insecure":  request.Settings.AllowInsecure,
					"update_interval": request.Settings.UpdateInterval,
					"active":          request.Settings.Active,
					"next_check":      time.Now().Add(time.Duration(request.Settings.UpdateInterval) * time.Minute),
				}

				if err := tx.Model(&existingServer).Updates(updates).Error; err != nil {
					response.Failed++
					response.Errors = append(response.Errors, "Failed to update server: "+cleanURL+" - "+err.Error())
					continue
				}

				// Queue the updated server for checking
				go checkServer(existingServer.ID, nil)

				response.Imported++ // Count updates as successful imports
			} else {
				// Skip existing servers when update_existing is false
				response.Failed++
				response.Errors = append(response.Errors, "Server URL already exists: "+cleanURL)
				continue
			}
		} else {
			// Create a new server
			server := Server{
				URL:                cleanURL,
				Active:             request.Settings.Active,
				FollowRedirect:     request.Settings.FollowRedirect,
				AllowInsecure:      request.Settings.AllowInsecure,
				ExpectedStatusCode: 200,
				UpdateInterval:     request.Settings.UpdateInterval,
				NextCheck:          time.Now().Add(time.Duration(request.Settings.UpdateInterval) * time.Minute),
			}

			// Attempt to create the server in the transaction
			if err := tx.Create(&server).Error; err != nil {
				response.Failed++
				response.Errors = append(response.Errors, "Failed to create server: "+cleanURL+" - "+err.Error())
				continue
			}

			// Queue the new server for checking
			go checkServer(server.ID, nil)

			response.Imported++
		}

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
