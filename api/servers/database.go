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
	var servers []Server

	now := time.Now()

	weekAgo := now.Add(-7 * 24 * time.Hour)
	if err := db.
		Preload("PingResults", func(db *gorm.DB) *gorm.DB {
			return db.Where("timestamp > ?", weekAgo).Order("timestamp DESC")
		}).
		Preload("PingResults.PingDetail").
		Where("active = ? AND next_check <= ?", true, now).
		Find(&servers).Error; err != nil {
		log.Printf("Error fetching servers: %v", err)
		return
	}

	for _, server := range servers {
		// Get the previous online status before pinging
		wasOnline := isServerOnline(server)

		// Get the previous certificate expiry status
		wasCertExpired, prevCertExpiry := getCertificateStatus(server)

		result := pingServer(server)

		interval, shouldRemainActive := calculateNextCheckInterval(server)

		server.NextCheck = now.Add(interval)
		wasActive := server.Active
		if !shouldRemainActive {
			server.Comment = fmt.Sprintf("WARNING: Server %s has had >95%% failures in past week. Automatically deactivated.", server.URL)
			server.Active = false
		}

		// Use a transaction for atomicity
		tx := db.Begin()
		if err := tx.Save(&server).Error; err != nil {
			tx.Rollback()
			log.Printf("Error saving server %s: %v", server.URL, err)
			continue
		}

		// Save PingDetail first if it exists
		if result.PingDetail != nil {
			if err := tx.Create(result.PingDetail).Error; err != nil {
				tx.Rollback()
				log.Printf("Error saving ping detail for %s: %v", server.URL, err)
				continue
			}
			result.DetailID = &result.PingDetail.ID
		}

		if err := tx.Create(&result).Error; err != nil {
			tx.Rollback()
			log.Printf("Error saving ping result for %s: %v", server.URL, err)
			continue
		}

		tx.Commit()

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
func getCertificateStatus(server Server) (bool, time.Time) {
	if len(server.PingResults) == 0 {
		return false, time.Time{}
	}

	var latest *PingResult
	for i := range server.PingResults {
		result := &server.PingResults[i]
		if result.PingDetail == nil || result.PingDetail.CertExpiryDate.IsZero() {
			continue
		}
		if latest == nil || result.Timestamp.After(latest.Timestamp) {
			latest = result
		}
	}

	if latest != nil {
		isExpired := time.Now().After(latest.PingDetail.CertExpiryDate)
		return isExpired, latest.PingDetail.CertExpiryDate
	}

	return false, time.Time{}
}
