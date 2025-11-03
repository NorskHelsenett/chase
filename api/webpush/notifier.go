package webpush

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// NotificationSender handles sending notifications to users
type NotificationSender struct {
	db              *gorm.DB
	vapidPublicKey  string
	vapidPrivateKey string
}

// NewNotificationSender creates a new notification sender
func NewNotificationSender(db *gorm.DB) (*NotificationSender, error) {
	keys, err := GetVAPIDKeys(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get VAPID keys: %v", err)
	}

	return &NotificationSender{
		db:              db,
		vapidPublicKey:  keys.PublicKey,
		vapidPrivateKey: keys.PrivateKey,
	}, nil
}

// SendToUser sends a notification to all of a user's subscriptions
func (ns *NotificationSender) SendToUser(userID uint, notification *Notification) error {
	subscriptions, err := GetUserSubscriptions(ns.db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user subscriptions: %v", err)
	}

	if len(subscriptions) == 0 {
		return fmt.Errorf("user has no subscriptions")
	}

	options := &SendOptions{
		TTL:             86400, // 24 hours
		Urgency:         "normal",
		VAPIDPublicKey:  ns.vapidPublicKey,
		VAPIDPrivateKey: ns.vapidPrivateKey,
	}

	var lastErr error
	successCount := 0

	for _, sub := range subscriptions {
		err := SendNotification(&sub, notification, options)
		if err != nil {
			log.Printf("Failed to send notification to subscription %d: %v", sub.ID, err)
			lastErr = err

			// If subscription is expired, clean it up
			if isSubscriptionExpiredError(err) {
				CleanupInvalidSubscriptions(ns.db, sub.Endpoint)
			}
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to send to any subscription: %v", lastErr)
	}

	return nil
}

// SendToAll sends a notification to all users subscribed to a specific event type
func (ns *NotificationSender) SendToAll(eventType NotificationEventType, notification *Notification, serverID *uint) (int, error) {
	subscriptions, err := GetSubscribersForEvent(ns.db, eventType)
	if err != nil {
		return 0, fmt.Errorf("failed to get subscribers: %v", err)
	}

	if len(subscriptions) == 0 {
		return 0, nil
	}

	options := &SendOptions{
		TTL:             86400, // 24 hours
		Urgency:         "normal",
		VAPIDPublicKey:  ns.vapidPublicKey,
		VAPIDPrivateKey: ns.vapidPrivateKey,
		Topic:           string(eventType),
	}

	successCount := 0
	for _, sub := range subscriptions {
		// Create notification log entry FIRST to get the ID
		logID, logErr := LogNotification(
			ns.db,
			sub.UserID,
			eventType,
			notification.Title,
			notification.Body,
			notification.URL,
			serverID,
			false, // Will update after send attempt
			"",
		)

		// Update notification to use the log ID in the URL
		notificationCopy := *notification
		if logID > 0 {
			notificationCopy.URL = fmt.Sprintf("/notification/%d", logID)
			if notificationCopy.Data == nil {
				notificationCopy.Data = make(map[string]interface{})
			}
			notificationCopy.Data["notificationId"] = logID
		}

		err := SendNotification(&sub, &notificationCopy, options)

		// Update the log entry with the result
		if logErr == nil && logID > 0 {
			ns.db.Model(&NotificationLog{}).Where("id = ?", logID).Updates(map[string]interface{}{
				"success":   err == nil,
				"error_msg": getErrorMessage(err),
				"sent_at":   time.Now(),
			})
		}

		if err != nil {
			// If subscription is expired, clean it up silently
			if isSubscriptionExpiredError(err) {
				CleanupInvalidSubscriptions(ns.db, sub.Endpoint)
			}
		} else {
			successCount++
		}
	}

	return successCount, nil
}

// NotifyServerAdded sends a notification when a new server is added
func (ns *NotificationSender) NotifyServerAdded(serverID uint, serverURL string) error {
	notification := &Notification{
		Title: "New Server Added",
		Body:  fmt.Sprintf("Server %s has been added to monitoring", serverURL),
		Icon:  "/icon-192.png",
		Tag:   "server-added",
		Data: map[string]interface{}{
			"type":      string(EventServerAdded),
			"serverUrl": serverURL,
			"serverId":  serverID,
			"url":       fmt.Sprintf("/server/%d", serverID),
			"timestamp": time.Now().Unix(),
		},
	}

	_, err := ns.SendToAll(EventServerAdded, notification, &serverID)
	return err
}

// NotifyServerOffline sends a notification when a server goes offline
func (ns *NotificationSender) NotifyServerOffline(serverID uint, serverURL, serverName string) error {
	notification := &Notification{
		Title: "Server Offline",
		Body:  fmt.Sprintf("%s is offline", serverName),
		Icon:  "/icon-192.png",
		Badge: "/badge-error.png",
		Tag:   "server-offline-" + serverURL,
		Data: map[string]interface{}{
			"type":      string(EventServerOffline),
			"serverUrl": serverURL,
			"serverId":  serverID,
			"url":       fmt.Sprintf("/server/%d", serverID),
			"timestamp": time.Now().Unix(),
		},
	}

	_, err := ns.SendToAll(EventServerOffline, notification, &serverID)
	return err
}

// NotifyServerOnline sends a notification when a server comes back online
func (ns *NotificationSender) NotifyServerOnline(serverID uint, serverURL, serverName string) error {
	notification := &Notification{
		Title: "Server Online",
		Body:  fmt.Sprintf("%s is back online", serverName),
		Icon:  "/icon-192.png",
		Badge: "/badge-success.png",
		Tag:   "server-online-" + serverURL,
		Data: map[string]interface{}{
			"type":      string(EventServerOnline),
			"serverUrl": serverURL,
			"serverId":  serverID,
			"url":       fmt.Sprintf("/server/%d", serverID),
			"timestamp": time.Now().Unix(),
		},
	}

	_, err := ns.SendToAll(EventServerOnline, notification, &serverID)
	return err
}

// NotifyServerDeleted sends a notification when a server is removed
func (ns *NotificationSender) NotifyServerDeleted(serverID uint, serverURL string) error {
	notification := &Notification{
		Title: "Server Removed",
		Body:  fmt.Sprintf("Server %s has been removed from monitoring", serverURL),
		Icon:  "/icon-192.png",
		Tag:   "server-deleted",
		Data: map[string]interface{}{
			"type":      string(EventServerDeleted),
			"serverUrl": serverURL,
			"serverId":  serverID,
			"timestamp": time.Now().Unix(),
		},
	}

	_, err := ns.SendToAll(EventServerDeleted, notification, &serverID)
	return err
}

// NotifyServerDeactivated sends a notification when a server is automatically deactivated
func (ns *NotificationSender) NotifyServerDeactivated(serverID uint, serverURL string, reason string) error {
	notification := &Notification{
		Title: "Server Deactivated",
		Body:  fmt.Sprintf("Server %s has been automatically deactivated: %s", serverURL, reason),
		Icon:  "/icon-192.png",
		Tag:   "server-deactivated",
		Data: map[string]interface{}{
			"type":      string(EventServerDeactivated),
			"serverUrl": serverURL,
			"serverId":  serverID,
			"reason":    reason,
			"timestamp": time.Now().Unix(),
		},
	}

	_, err := ns.SendToAll(EventServerDeactivated, notification, &serverID)
	return err
}

// NotifyScanCompleted sends a notification when a security scan completes
func (ns *NotificationSender) NotifyScanCompleted(serverID uint, serverURL string, findingsCount int) error {
	notification := &Notification{
		Title: "Security Scan Complete",
		Body:  fmt.Sprintf("Scan of %s found %d findings", serverURL, findingsCount),
		Icon:  "/icon-192.png",
		Tag:   "scan-complete-" + serverURL,
		Data: map[string]interface{}{
			"type":          string(EventScanCompleted),
			"serverUrl":     serverURL,
			"serverId":      serverID,
			"findingsCount": findingsCount,
			"url":           fmt.Sprintf("/server/%d", serverID),
			"timestamp":     time.Now().Unix(),
		},
	}

	_, err := ns.SendToAll(EventScanCompleted, notification, &serverID)
	return err
}

// NotifyHighRiskFound sends a notification when high/critical risks are detected
func (ns *NotificationSender) NotifyHighRiskFound(serverID uint, serverURL string, riskLevel, description string) error {
	notification := &Notification{
		Title: fmt.Sprintf("%s Risk Detected", riskLevel),
		Body:  fmt.Sprintf("%s: %s", serverURL, description),
		Icon:  "/icon-192.png",
		Badge: "/badge-warning.png",
		Tag:   "high-risk-" + serverURL,
		Data: map[string]interface{}{
			"type":        string(EventHighRiskFound),
			"serverUrl":   serverURL,
			"serverId":    serverID,
			"riskLevel":   riskLevel,
			"description": description,
			"url":         fmt.Sprintf("/server/%d", serverID),
			"timestamp":   time.Now().Unix(),
		},
		Actions: []NotificationAction{
			{
				Action: "view",
				Title:  "View Details",
			},
			{
				Action: "dismiss",
				Title:  "Dismiss",
			},
		},
	}

	_, err := ns.SendToAll(EventHighRiskFound, notification, &serverID)
	return err
}

// Helper functions

func isSubscriptionExpiredError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "410") || contains(errStr, "404") || contains(errStr, "expired")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
