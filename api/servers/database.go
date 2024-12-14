package servers

import (
	"log"
	"time"

	"github.com/norskhelsenett/chase/database"
)

func StartMonitoring() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	// Run immediately in a goroutine
	go runMonitoring()

	// Then run on ticker
	for range ticker.C {
		go runMonitoring()
	}
}

func runMonitoring() {
	db := database.GetDB()
	var servers []Server

	now := time.Now()

	if err := db.Where("active = ? AND next_check <= ?", true, now).Find(&servers).Error; err != nil {
		log.Printf("Error fetching servers: %v", err)
		return
	}

	for _, server := range servers {

		if !server.Active {
			continue
		}

		result := pingServer(server)

		nextCheck := calculateNextCheckInterval(server.FailureCount)
		server.NextCheck = now.Add(nextCheck)

		db.Save(&server)
		db.Create(&result)
	}
}
