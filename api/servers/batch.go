// BatchImportServers handles importing multiple servers at once
package servers

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/utils"
	"gorm.io/gorm"
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
		request.Settings.UpdateInterval = 15
	}

	// Collect IDs to schedule after commit
	var serversToCheck []uint

	for _, site := range request.Sites {
		formattedURL, err := utils.EnsureHTTPS(site)
		if err != nil {
			response.Failed++
			response.Errors = append(response.Errors, "Invalid URL format: "+site)
			continue
		}

		parsedURL, err := url.Parse(formattedURL)
		if err != nil || parsedURL.Host == "" {
			response.Failed++
			response.Errors = append(response.Errors, "Invalid URL format: "+site)
			continue
		}

		cleanURL := strings.TrimPrefix(strings.TrimPrefix(formattedURL, "https://"), "http://")

		// Try to fetch existing server (safe, returns only one row)
		var existing Server
		lookupErr := tx.Unscoped().
			Where("url = ?", cleanURL).
			Take(&existing).
			Error

		// If exists
		if lookupErr == nil {
			if request.UpdateExisting {
				// Undelete if soft deleted
				if existing.DeletedAt.Valid {
					if err := tx.Unscoped().
						Model(&existing).
						Update("deleted_at", nil).Error; err != nil {
						response.Failed++
						response.Errors = append(response.Errors, "Failed to restore deleted: "+cleanURL)
						continue
					}
				}

				updates := map[string]interface{}{
					"follow_redirect": request.Settings.FollowRedirect,
					"allow_insecure":  request.Settings.AllowInsecure,
					"update_interval": request.Settings.UpdateInterval,
					"active":          request.Settings.Active,
					"next_check":      time.Now().Add(time.Duration(request.Settings.UpdateInterval) * time.Minute),
				}

				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					response.Failed++
					response.Errors = append(response.Errors, "Failed to update server: "+cleanURL)
					continue
				}

				response.Imported++
				serversToCheck = append(serversToCheck, existing.ID)
			} else {
				response.Failed++
				response.Errors = append(response.Errors, "Server already exists: "+cleanURL)
			}

			continue
		}

		// If record not found → create new server
		if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			// unexpected DB error
			response.Failed++
			response.Errors = append(response.Errors, "DB error checking server: "+cleanURL)
			continue
		}

		// Create new server
		server := Server{
			URL:                cleanURL,
			Active:             request.Settings.Active,
			FollowRedirect:     request.Settings.FollowRedirect,
			AllowInsecure:      request.Settings.AllowInsecure,
			ExpectedStatusCode: 200,
			UpdateInterval:     request.Settings.UpdateInterval,
			NextCheck:          time.Now().Add(time.Duration(request.Settings.UpdateInterval) * time.Minute),
		}

		if err := tx.Create(&server).Error; err != nil {
			response.Failed++
			response.Errors = append(response.Errors, "Failed to create server: "+cleanURL)
			continue
		}

		response.Imported++
		serversToCheck = append(serversToCheck, server.ID)
	}

	// Commit if anything changed
	if response.Imported > 0 {
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Failed to commit transaction"})
			return
		}
	} else {
		tx.Rollback()
	}

	// Schedule checks *after* commit (never inside transaction)
	for _, id := range serversToCheck {
		go checkServer(id, nil)
	}

	// Tell connected clients to refetch so the imported servers show up live.
	// One signal for the whole batch — a per-server flood would overrun the
	// client buffers and get dropped.
	if response.Imported > 0 {
		BroadcastServersChanged()
	}

	c.JSON(200, response)
}
