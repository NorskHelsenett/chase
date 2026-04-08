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

type pingSummary struct {
	StatusCode   int         `json:"status_code"`
	ResponseTime float64     `json:"response_time_ms"`
	Error        string      `json:"error,omitempty"`
	Timestamp    time.Time   `json:"timestamp"`
	Detail       *PingDetail `json:"detail,omitempty"`
}

type serverSummary struct {
	ID                  uint      `json:"ID"`
	URL                 string    `json:"url"`
	Active              bool      `json:"active"`
	FollowRedirect      bool      `json:"follow_redirect"`
	NextCheck           time.Time `json:"next_check"`
	AllowInsecure       bool      `json:"allow_insecure"`
	ExpectedStatusCode  int       `json:"expected_status"`
	Comment             string    `json:"comment"`
	UpdateInterval      int       `json:"update_interval"`
	CreatedAt           time.Time `json:"CreatedAt"`
	Status              string    `json:"status"`
	LastStatusCode      int       `json:"last_status_code,omitempty"`
	LastPingTime        time.Time `json:"last_ping_time,omitempty"`
	LastResponseMs      float64   `json:"last_response_ms,omitempty"`
	SecurityRiskLevel   string    `json:"security_risk_level,omitempty"`
	SecurityDescription string    `json:"security_description,omitempty"`
	SecurityScanTime    time.Time `json:"security_scan_time,omitempty"`
	HeaderScore         string    `json:"header_score,omitempty"`
	CertScore           string    `json:"cert_score,omitempty"`
	AdminRisk           string    `json:"admin_risk,omitempty"`
	APIRisk             string    `json:"api_risk,omitempty"`
	SecretsRisk         string    `json:"secrets_risk,omitempty"`
	SecretsCount        int       `json:"secrets_count,omitempty"`
	SiteTitle           string    `json:"site_title,omitempty"`
	SiteDescription     string    `json:"site_description,omitempty"`
	OGImage             string    `json:"og_image,omitempty"`
	Favicon             string    `json:"favicon,omitempty"`
}

// GetServerResults handles the request to retrieve ping results for a specific server.
//
// It extracts the server ID from the URL parameters and validates it exists in the database.
// The function supports the following query parameters:
//   - limit: Maximum number of results to return (default: 20, max: 100)
//   - sort: Sort order for results by timestamp ("asc" or "desc", default: "desc")
//   - from: Start time for results filtering (RFC3339 format, default: 7 days ago)
//   - to: End time for results filtering (RFC3339 format, default: current time)
//
// Returns:
//   - 200 OK with array of ping results on success
//   - 400 Bad Request if server ID is missing or time parameters are invalid
//   - 404 Not Found if the server doesn't exist
//   - 500 Internal Server Error if database query fails
func GetServerResults(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Server ID is required"})
		return
	}

	db := database.GetDB()

	var server Server
	if err := db.First(&server, serverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	// --- Parse query parameters ---
	// Limit
	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				limit = 100
			} else {
				limit = parsedLimit
			}
		}
	}

	// Sort
	sortOrder := "DESC"
	if strings.ToLower(c.Query("sort")) == "asc" {
		sortOrder = "ASC"
	}

	// includeDetail
	includeDetail := false
	if includeDetailParam := c.Query("includeDetail"); strings.ToLower(includeDetailParam) == "true" {
		includeDetail = true
	}

	// From/To or Range
	var fromTime, toTime time.Time
	var err error

	if fromParam := c.Query("from"); fromParam != "" {
		fromTime, err = time.Parse(time.RFC3339, fromParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' time. Use RFC3339 format"})
			return
		}
	} else {
		rangeHours := 336 // default 2 weeks
		if rangeParam := c.Query("range"); rangeParam != "" {
			if parsedRange, err := strconv.Atoi(rangeParam); err == nil && parsedRange > 0 {
				if parsedRange > 24*90 {
					rangeHours = 24 * 90
				} else {
					rangeHours = parsedRange
				}
			}
		}
		fromTime = time.Now().Add(-time.Duration(rangeHours) * time.Hour)
	}

	if toParam := c.Query("to"); toParam != "" {
		toTime, err = time.Parse(time.RFC3339, toParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' time. Use RFC3339 format"})
			return
		}
	} else {
		toTime = time.Now()
	}

	// --- Build the query ---
	query := db.Where("server_id = ? AND timestamp BETWEEN ? AND ?", serverID, fromTime, toTime).
		Order("timestamp " + sortOrder).
		Limit(limit)

	if includeDetail {
		query = query.Preload("PingDetail")
	}

	var results []PingResult
	if err := query.Find(&results).Error; err != nil {
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

	summaries := make([]serverSummary, len(servers))

	for i := range servers {
		summaries[i] = serverSummary{
			ID:                 servers[i].ID,
			URL:                servers[i].URL,
			Active:             servers[i].Active,
			FollowRedirect:     servers[i].FollowRedirect,
			NextCheck:          servers[i].NextCheck,
			AllowInsecure:      servers[i].AllowInsecure,
			ExpectedStatusCode: servers[i].ExpectedStatusCode,
			Comment:            servers[i].Comment,
			UpdateInterval:     servers[i].UpdateInterval,
			CreatedAt:          servers[i].CreatedAt,
			Status:             "unknown",
			Favicon:            servers[i].Favicon,
			SiteTitle:          servers[i].SiteTitle,
			SiteDescription:    servers[i].SiteDescription,
			OGImage:            servers[i].OGImage,
		}

		// Check latest ping to determine status
		var latest PingResult
		if err := db.Model(&PingResult{}).
			Select("status_code, response_time, error, timestamp").
			Where("server_id = ?", servers[i].ID).
			Order("timestamp DESC").
			First(&latest).Error; err == nil {

			// Consider the ping stale if it's older than 2x the update interval
			staleCutoff := time.Now().Add(-time.Duration(servers[i].UpdateInterval*2) * time.Minute)
			if latest.Timestamp.Before(staleCutoff) {
				summaries[i].Status = "stale"
			} else if latest.Error != "" {
				summaries[i].Status = "down"
			} else if latest.StatusCode == servers[i].ExpectedStatusCode {
				summaries[i].Status = "up"
			} else {
				summaries[i].Status = "down"
			}

			summaries[i].LastStatusCode = latest.StatusCode
			summaries[i].LastPingTime = latest.Timestamp
			summaries[i].LastResponseMs = latest.ResponseTime
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
			summaries[i].SecurityRiskLevel = securityReport.RiskLevel
			summaries[i].SecurityDescription = securityReport.Description
			summaries[i].SecurityScanTime = securityReport.CreatedAt

			// Extract additional security details from report data if available
			if len(securityReport.ReportData) > 0 {
				var fullReport struct {
					Headers struct {
						Score string `json:"score"`
						Title string `json:"title"`
						Risk  string `json:"risk"`
						Meta  struct {
							Description string `json:"description"`
							OGImage     string `json:"og_image"`
							Favicon     string `json:"favicon"`
						} `json:"meta"`
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
					SecretExposure struct {
						Risk     string `json:"risk"`
						Findings []struct {
							Description string `json:"description"`
						} `json:"findings"`
					} `json:"secretExposure"`
				}

				if err := json.Unmarshal(securityReport.ReportData, &fullReport); err == nil {
					summaries[i].HeaderScore = fullReport.Headers.Score
					summaries[i].CertScore = fullReport.Certificate.Grade
					summaries[i].AdminRisk = string(fullReport.AdminPages.Risk)
					summaries[i].APIRisk = string(fullReport.Swagger.Risk)
					summaries[i].SecretsRisk = string(fullReport.SecretExposure.Risk)
					summaries[i].SecretsCount = len(fullReport.SecretExposure.Findings)
					if fullReport.Headers.Title != "" {
						summaries[i].SiteTitle = fullReport.Headers.Title
					}
					if fullReport.Headers.Meta.Description != "" {
						summaries[i].SiteDescription = fullReport.Headers.Meta.Description
					}
					if fullReport.Headers.Meta.OGImage != "" {
						summaries[i].OGImage = fullReport.Headers.Meta.OGImage
					}
					if fullReport.Headers.Meta.Favicon != "" {
						summaries[i].Favicon = fullReport.Headers.Meta.Favicon
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, summaries)
}
