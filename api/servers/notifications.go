package servers

import (
	"log"

	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/webpush"
)

// NotifyServerAdded sends a push notification when a new server is added
func NotifyServerAdded(serverURL string) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifyServerAdded(serverURL); err != nil {
		log.Printf("Failed to send server added notification: %v", err)
	}
}

// NotifyServerStatusChange sends notifications when server status changes
func NotifyServerStatusChange(serverURL, serverName string, wasOnline, isOnline bool) {
	// Only notify on actual status changes
	if wasOnline == isOnline {
		return
	}

	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if isOnline {
		if err := sender.NotifyServerOnline(serverURL, serverName); err != nil {
			log.Printf("Failed to send server online notification: %v", err)
		}
	} else {
		if err := sender.NotifyServerOffline(serverURL, serverName); err != nil {
			log.Printf("Failed to send server offline notification: %v", err)
		}
	}
}

// NotifyServerDeleted sends a push notification when a server is removed
func NotifyServerDeleted(serverURL string) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifyServerDeleted(serverURL); err != nil {
		log.Printf("Failed to send server deleted notification: %v", err)
	}
}

// NotifyServerDeactivated sends a push notification when a server is automatically deactivated
func NotifyServerDeactivated(serverURL, reason string) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifyServerDeactivated(serverURL, reason); err != nil {
		log.Printf("Failed to send server deactivated notification: %v", err)
	}
}
