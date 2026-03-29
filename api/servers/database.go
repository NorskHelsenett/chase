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
	if err := db.AutoMigrate(&Server{}, &PingResult{}); err != nil {
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
