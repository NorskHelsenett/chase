package servers

import (
	"fmt"
	"log"
	"time"

	"github.com/norskhelsenett/chase/database"
)

func StartMonitoring() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	go runMonitoring()

	for range ticker.C {
		go runMonitoring()
	}
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
