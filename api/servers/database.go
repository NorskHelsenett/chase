package servers

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/norskhelsenett/chase/database"
	"gorm.io/gorm"
)

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
	db := database.GetDB()
	var servers []Server

	now := time.Now()

	weekAgo := now.Add(-7 * 24 * time.Hour)
	if err := db.Preload("PingResults", "timestamp > ?", weekAgo).
		Where("active = ? AND next_check <= ?", true, now).
		Find(&servers).Error; err != nil {
		log.Printf("Error fetching servers: %v", err)
		return
	}

	for _, server := range servers {
		result := pingServer(server)

		interval, shouldRemainActive := calculateNextCheckInterval(server)

		server.NextCheck = now.Add(interval)
		if !shouldRemainActive {
			log.Printf("WARNING: Server %s has had >95%% failures in past week. Consider deactivating.", server.URL)
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

		if err := tx.Create(&result).Error; err != nil {
			tx.Rollback()
			log.Printf("Error saving ping result for %s: %v", server.URL, err)
			continue
		}

		tx.Commit()
	}
}
