package servers

import (
	"log"
	"time"

	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/webpush"
)

// NotifyServerAdded sends a push notification when a new server is added
func NotifyServerAdded(serverID uint, serverURL string) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifyServerAdded(serverID, serverURL); err != nil {
		log.Printf("Failed to send server added notification: %v", err)
	}
}

// NotifyServerStatusChange sends notifications when server status changes
func NotifyServerStatusChange(serverID uint, serverURL, serverName string, wasOnline, isOnline bool) {
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
		if err := sender.NotifyServerOnline(serverID, serverURL, serverName); err != nil {
			log.Printf("Failed to send server online notification: %v", err)
		}
	} else {
		if err := sender.NotifyServerOffline(serverID, serverURL, serverName); err != nil {
			log.Printf("Failed to send server offline notification: %v", err)
		}
	}
}

// NotifyServerDeleted sends a push notification when a server is removed
func NotifyServerDeleted(serverID uint, serverURL string) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifyServerDeleted(serverID, serverURL); err != nil {
		log.Printf("Failed to send server deleted notification: %v", err)
	}
}

// NotifyServerDeactivated sends a push notification when a server is automatically deactivated
func NotifyServerDeactivated(serverID uint, serverURL, reason string) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifyServerDeactivated(serverID, serverURL, reason); err != nil {
		log.Printf("Failed to send server deactivated notification: %v", err)
	}
}

// NotifyCertificateExpired sends a push notification when a certificate has expired
func NotifyCertificateExpired(serverID uint, serverURL, serverName string, expiryDate time.Time) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifyCertificateExpired(serverID, serverURL, serverName, expiryDate); err != nil {
		log.Printf("Failed to send certificate expired notification: %v", err)
	}
}

// NotifyCertificateExpiringSoon sends a push notification when a certificate is expiring soon
func NotifyCertificateExpiringSoon(serverID uint, serverURL, serverName string, expiryDate time.Time, daysLeft int) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifyCertificateExpiringSoon(serverID, serverURL, serverName, expiryDate, daysLeft); err != nil {
		log.Printf("Failed to send certificate expiring soon notification: %v", err)
	}
}

// NotifySecurityTxtExpired sends a push notification when security.txt has expired
func NotifySecurityTxtExpired(serverID uint, serverURL, serverName string, expiryDate time.Time) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifySecurityTxtExpired(serverID, serverURL, serverName, expiryDate); err != nil {
		log.Printf("Failed to send security.txt expired notification: %v", err)
	}
}

// NotifySecurityTxtExpiring7Days sends a push notification when security.txt expires in 7 days or less
func NotifySecurityTxtExpiring7Days(serverID uint, serverURL, serverName string, expiryDate time.Time, daysLeft int) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifySecurityTxtExpiring7Days(serverID, serverURL, serverName, expiryDate, daysLeft); err != nil {
		log.Printf("Failed to send security.txt expiring soon (7 days) notification: %v", err)
	}
}

// NotifySecurityTxtExpiring90Days sends a push notification when security.txt expires in 90 days or less
func NotifySecurityTxtExpiring90Days(serverID uint, serverURL, serverName string, expiryDate time.Time, daysLeft int) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifySecurityTxtExpiring90Days(serverID, serverURL, serverName, expiryDate, daysLeft); err != nil {
		log.Printf("Failed to send security.txt expiring soon (90 days) notification: %v", err)
	}
}

// NotifySecurityTxtExpiring30Days sends a push notification when security.txt expires in 30 days or less
func NotifySecurityTxtExpiring30Days(serverID uint, serverURL, serverName string, expiryDate time.Time, daysLeft int) {
	db := database.GetDB()
	sender, err := webpush.NewNotificationSender(db)
	if err != nil {
		log.Printf("Failed to create notification sender: %v", err)
		return
	}

	if err := sender.NotifySecurityTxtExpiring30Days(serverID, serverURL, serverName, expiryDate, daysLeft); err != nil {
		log.Printf("Failed to send security.txt expiring soon (30 days) notification: %v", err)
	}
}
