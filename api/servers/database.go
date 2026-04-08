package servers

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/norskhelsenett/chase/database"
	"gorm.io/gorm"
)

var monitoringInProgress atomic.Bool

const serverBatchSize = 100

func AutoMigrate(db *gorm.DB) error {
	// Migrate the schemas
	if err := db.AutoMigrate(&Server{}, &PingResult{}, &PingHourlySummary{}, &PingDailySummary{}, &GeoCache{}); err != nil {
		return err
	}

	// Create composite indexes if they don't exist
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_active_next_check ON servers (active, next_check);
		CREATE INDEX IF NOT EXISTS idx_server_timestamp ON ping_results (server_id, timestamp);
	`).Error; err != nil {
		return err
	}

	return nil
}

// AggregateAndPrunePings runs the three-tier ping retention cleanup.
// Call from a single background goroutine to avoid SQLite lock contention.
func AggregateAndPrunePings() {
	aggregateAndPrunePings(database.GetDB())
}

type aggRow struct {
	ServerID        uint
	Bucket          string
	Total           int
	Successful      int
	Failed          int
	AvgResponseTime float64
	MinResponseTime float64
	MaxResponseTime float64
}

// aggregateAndPrunePings implements a three-tier retention policy.
// Runs in a goroutine — must not crash the process.
//
//	Last 7 days  → every raw ping kept
//	7–30 days    → aggregated to hourly, raw pings deleted
//	30+ days     → aggregated to daily, hourly summaries deleted
func aggregateAndPrunePings(db *gorm.DB) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Ping aggregation recovered from panic: %v", r)
		}
	}()

	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, -1, 0)

	// --- Tier 2: raw pings 7–30 days old → hourly summaries ---
	var hourlyRows []aggRow
	db.Raw(`
		SELECT server_id,
			strftime('%Y-%m-%d %H:00:00', timestamp) as bucket,
			COUNT(*) as total,
			SUM(CASE WHEN error = '' THEN 1 ELSE 0 END) as successful,
			SUM(CASE WHEN error != '' THEN 1 ELSE 0 END) as failed,
			AVG(CASE WHEN error = '' THEN response_time ELSE NULL END) as avg_response_time,
			MIN(CASE WHEN error = '' THEN response_time ELSE NULL END) as min_response_time,
			MAX(CASE WHEN error = '' THEN response_time ELSE NULL END) as max_response_time
		FROM ping_results
		WHERE timestamp < ? AND timestamp >= ? AND deleted_at IS NULL
		GROUP BY server_id, strftime('%Y-%m-%d %H', timestamp)
	`, weekAgo, monthAgo).Scan(&hourlyRows)

	if len(hourlyRows) > 0 {
		log.Printf("Aggregating %d hourly ping summaries (7-30 days)", len(hourlyRows))
		for _, r := range hourlyRows {
			hour, _ := time.Parse("2006-01-02 15:04:05", r.Bucket)
			var existing PingHourlySummary
			if err := db.Where("server_id = ? AND hour = ?", r.ServerID, hour).First(&existing).Error; err == nil {
				db.Model(&existing).Updates(map[string]interface{}{
					"total": r.Total, "successful": r.Successful, "failed": r.Failed,
					"avg_response_time": r.AvgResponseTime, "min_response_time": r.MinResponseTime, "max_response_time": r.MaxResponseTime,
				})
			} else {
				db.Create(&PingHourlySummary{
					ServerID: r.ServerID, Hour: hour, Total: r.Total, Successful: r.Successful, Failed: r.Failed,
					AvgResponseTime: r.AvgResponseTime, MinResponseTime: r.MinResponseTime, MaxResponseTime: r.MaxResponseTime,
				})
			}
		}

		// Delete raw pings that are now aggregated (7-30 days old)
		result := db.Unscoped().Where("timestamp < ? AND timestamp >= ?", weekAgo, monthAgo).Delete(&PingResult{})
		log.Printf("Pruned %d raw pings (7-30 days old)", result.RowsAffected)
	}

	// --- Tier 3: hourly summaries older than 30 days → daily summaries ---
	var dailyRows []aggRow
	db.Raw(`
		SELECT server_id,
			DATE(hour) as bucket,
			SUM(total) as total,
			SUM(successful) as successful,
			SUM(failed) as failed,
			SUM(avg_response_time * total) / NULLIF(SUM(total), 0) as avg_response_time,
			MIN(min_response_time) as min_response_time,
			MAX(max_response_time) as max_response_time
		FROM ping_hourly_summaries
		WHERE hour < ?
		GROUP BY server_id, DATE(hour)
	`, monthAgo).Scan(&dailyRows)

	if len(dailyRows) > 0 {
		log.Printf("Aggregating %d daily ping summaries (30+ days)", len(dailyRows))
		for _, r := range dailyRows {
			date, _ := time.Parse("2006-01-02", r.Bucket)
			var existing PingDailySummary
			if err := db.Where("server_id = ? AND date = ?", r.ServerID, date).First(&existing).Error; err == nil {
				db.Model(&existing).Updates(map[string]interface{}{
					"total": r.Total, "successful": r.Successful, "failed": r.Failed,
					"avg_response_time": r.AvgResponseTime, "min_response_time": r.MinResponseTime, "max_response_time": r.MaxResponseTime,
				})
			} else {
				db.Create(&PingDailySummary{
					ServerID: r.ServerID, Date: date, Total: r.Total, Successful: r.Successful, Failed: r.Failed,
					AvgResponseTime: r.AvgResponseTime, MinResponseTime: r.MinResponseTime, MaxResponseTime: r.MaxResponseTime,
				})
			}
		}

		// Delete hourly summaries now rolled into daily
		result := db.Where("hour < ?", monthAgo).Delete(&PingHourlySummary{})
		log.Printf("Pruned %d hourly summaries (30+ days old)", result.RowsAffected)
	}

	// --- Also delete any raw pings older than 30 days (in case they were missed) ---
	result := db.Unscoped().Where("timestamp < ?", monthAgo).Delete(&PingResult{})
	if result.RowsAffected > 0 {
		log.Printf("Pruned %d raw pings older than 30 days", result.RowsAffected)
	}

	// --- Clean up orphaned ping_details ---
	orphaned := db.Exec(`DELETE FROM ping_details WHERE id NOT IN (SELECT DISTINCT detail_id FROM ping_results WHERE detail_id IS NOT NULL)`)
	if orphaned.RowsAffected > 0 {
		log.Printf("Pruned %d orphaned ping details", orphaned.RowsAffected)
	}

	log.Printf("Ping aggregation complete")
}

func StartMonitoring() {
	interval := getMonitoringInterval()
	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	defer ticker.Stop()

	go runMonitoring()

	for range ticker.C {
		go runMonitoring()
	}
}

func getMonitoringInterval() int {
	intervalStr := os.Getenv("MONITORING_INTERVAL")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		return 1
	}
	return interval
}

func runMonitoring() {
	if !monitoringInProgress.CompareAndSwap(false, true) {
		log.Printf("Monitoring run already in progress, skipping")
		return
	}
	defer monitoringInProgress.Store(false)

	db := database.GetDB()
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)

	var servers []Server
	if err := db.
		Where("active = ? AND next_check <= ?", true, now).
		FindInBatches(&servers, serverBatchSize, func(tx *gorm.DB, batch int) error {
			for _, server := range servers {
				var recentResults []PingResult
				if err := db.Where("server_id = ? AND timestamp > ?", server.ID, weekAgo).
					Order("timestamp DESC").
					Find(&recentResults).Error; err != nil {
					log.Printf("Error fetching ping history for %s: %v", server.URL, err)
					recentResults = nil
				}
				server.PingResults = recentResults

				// Get the previous online status before pinging
				wasOnline := isServerOnline(server)

				// Get the previous certificate expiry status
				wasCertExpired, prevCertExpiry := getCertificateStatus(db, server.ID)

				result := pingServer(server)

				// Update site metadata if extracted during ping
				if result.siteMetadata.Favicon != "" {
					server.Favicon = result.siteMetadata.Favicon
				}
				if result.siteMetadata.Title != "" {
					server.SiteTitle = result.siteMetadata.Title
				}
				if result.siteMetadata.Description != "" {
					server.SiteDescription = result.siteMetadata.Description
				}
				if result.siteMetadata.OGImage != "" {
					server.OGImage = result.siteMetadata.OGImage
				}

				interval, shouldRemainActive := calculateNextCheckInterval(server)

				server.NextCheck = now.Add(interval)
				wasActive := server.Active
				if !shouldRemainActive {
					server.Comment = fmt.Sprintf("WARNING: Server %s has had >95%% failures in past week. Automatically deactivated.", server.URL)
					server.Active = false
				}

				// Use a transaction for atomicity
				txn := db.Begin()
				if err := txn.Save(&server).Error; err != nil {
					txn.Rollback()
					log.Printf("Error saving server %s: %v", server.URL, err)
					continue
				}

				// Save PingDetail first if it exists
				if result.PingDetail != nil {
					if err := txn.Create(result.PingDetail).Error; err != nil {
						txn.Rollback()
						log.Printf("Error saving ping detail for %s: %v", server.URL, err)
						continue
					}
					result.DetailID = &result.PingDetail.ID
				}

				if err := txn.Create(&result).Error; err != nil {
					txn.Rollback()
					log.Printf("Error saving ping result for %s: %v", server.URL, err)
					continue
				}

				txn.Commit()

				// Broadcast ping result to SSE clients
				BroadcastPing(server.ID, server.ExpectedStatusCode, result)

				// Send notification if server was deactivated
				if wasActive && !server.Active {
					NotifyServerDeactivated(server.ID, server.URL, ">95% failures in past week")
				}

				// Check if status changed and send notification
				isOnline := result.Error == ""
				if wasOnline != isOnline {
					serverName := server.URL
					if server.Comment != "" && len(server.Comment) < 100 {
						serverName = server.Comment
					}
					go NotifyServerStatusChange(server.ID, server.URL, serverName, wasOnline, isOnline)
				}

				// Check certificate expiry status and send notification
				if result.PingDetail != nil {
					isCertExpired := !result.PingDetail.CertExpiryDate.IsZero() && time.Now().After(result.PingDetail.CertExpiryDate)
					daysUntilExpiry := int(time.Until(result.PingDetail.CertExpiryDate).Hours() / 24)

					// Notify if certificate just expired
					if isCertExpired && !wasCertExpired {
						serverName := server.URL
						if server.Comment != "" && len(server.Comment) < 100 {
							serverName = server.Comment
						}
						go NotifyCertificateExpired(server.ID, server.URL, serverName, result.PingDetail.CertExpiryDate)
					} else if !isCertExpired && daysUntilExpiry <= 14 && daysUntilExpiry > 0 {
						// Notify for certificates expiring soon (only if we haven't notified before or if the expiry date changed)
						if prevCertExpiry.IsZero() || !prevCertExpiry.Equal(result.PingDetail.CertExpiryDate) {
							serverName := server.URL
							if server.Comment != "" && len(server.Comment) < 100 {
								serverName = server.Comment
							}
							go NotifyCertificateExpiringSoon(server.ID, server.URL, serverName, result.PingDetail.CertExpiryDate, daysUntilExpiry)
						}
					}
				}
			}
			return nil
		}).Error; err != nil {
		log.Printf("Error fetching servers: %v", err)
		return
	}
}

// isServerOnline checks if the most recent ping was successful
func isServerOnline(server Server) bool {
	if len(server.PingResults) == 0 {
		return true // Assume online if no results yet
	}

	// Get the most recent result
	mostRecent := server.PingResults[0]
	for _, result := range server.PingResults {
		if result.Timestamp.After(mostRecent.Timestamp) {
			mostRecent = result
		}
	}

	return mostRecent.Error == ""
}

// getCertificateStatus checks the previous certificate expiry status
// Returns (isExpired, expiryDate)
func getCertificateStatus(db *gorm.DB, serverID uint) (bool, time.Time) {
	var latest PingResult
	if err := db.
		Where("server_id = ? AND detail_id IS NOT NULL", serverID).
		Preload("PingDetail").
		Order("timestamp DESC").
		First(&latest).Error; err != nil || latest.PingDetail == nil {
		return false, time.Time{}
	}

	isExpired := time.Now().After(latest.PingDetail.CertExpiryDate)
	return isExpired, latest.PingDetail.CertExpiryDate
}
