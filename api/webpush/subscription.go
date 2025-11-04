package webpush

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// SubscribeUser creates or updates a push subscription for a user
func SubscribeUser(db *gorm.DB, userID uint, endpoint, auth, p256dh string) error {
	if endpoint == "" || auth == "" || p256dh == "" {
		return errors.New("endpoint, auth, and p256dh are required")
	}

	// Check if subscription already exists
	var existing PushSubscription
	err := db.Where("endpoint = ?", endpoint).First(&existing).Error

	if err == nil {
		// Update existing subscription
		existing.UserID = userID
		existing.Auth = auth
		existing.P256dh = p256dh
		return db.Save(&existing).Error
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Create new subscription
	subscription := PushSubscription{
		UserID:   userID,
		Endpoint: endpoint,
		Auth:     auth,
		P256dh:   p256dh,
	}

	return db.Create(&subscription).Error
}

// UnsubscribeUser removes a push subscription
func UnsubscribeUser(db *gorm.DB, userID uint, endpoint string) error {
	return db.Where("user_id = ? AND endpoint = ?", userID, endpoint).Delete(&PushSubscription{}).Error
}

// UnsubscribeAllForUser removes all push subscriptions for a user
func UnsubscribeAllForUser(db *gorm.DB, userID uint) error {
	return db.Where("user_id = ?", userID).Delete(&PushSubscription{}).Error
}

// GetUserSubscriptions retrieves all push subscriptions for a user
func GetUserSubscriptions(db *gorm.DB, userID uint) ([]PushSubscription, error) {
	var subscriptions []PushSubscription
	err := db.Where("user_id = ?", userID).Find(&subscriptions).Error
	return subscriptions, err
}

// GetAllSubscriptions retrieves all active push subscriptions
func GetAllSubscriptions(db *gorm.DB) ([]PushSubscription, error) {
	var subscriptions []PushSubscription
	err := db.Find(&subscriptions).Error
	return subscriptions, err
}

// SetNotificationPreference sets whether a user wants to receive a specific notification type
func SetNotificationPreference(db *gorm.DB, userID uint, eventType NotificationEventType, enabled bool) error {
	var pref NotificationPreference
	err := db.Where("user_id = ? AND event_type = ?", userID, eventType).First(&pref).Error

	if err == nil {
		// Update existing preference
		pref.Enabled = enabled
		return db.Save(&pref).Error
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Create new preference
	pref = NotificationPreference{
		UserID:    userID,
		EventType: eventType,
		Enabled:   enabled,
	}

	return db.Create(&pref).Error
}

// GetNotificationPreferences retrieves all notification preferences for a user
func GetNotificationPreferences(db *gorm.DB, userID uint) (map[NotificationEventType]bool, error) {
	var prefs []NotificationPreference
	if err := db.Where("user_id = ?", userID).Find(&prefs).Error; err != nil {
		return nil, err
	}

	result := make(map[NotificationEventType]bool)

	// Set defaults for all event types
	for _, eventType := range GetAllEventTypes() {
		result[eventType] = true // Default to enabled
	}

	// Override with user preferences
	for _, pref := range prefs {
		result[pref.EventType] = pref.Enabled
	}

	return result, nil
}

// IsUserSubscribedToEvent checks if a user wants to receive a specific event type
func IsUserSubscribedToEvent(db *gorm.DB, userID uint, eventType NotificationEventType) bool {
	var pref NotificationPreference
	err := db.Where("user_id = ? AND event_type = ?", userID, eventType).First(&pref).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true // Default to enabled if no preference set
	}

	if err != nil {
		log.Printf("Error checking notification preference: %v", err)
		return false
	}

	return pref.Enabled
}

// GetSubscribersForEvent returns all subscriptions for users who want to receive this event type
func GetSubscribersForEvent(db *gorm.DB, eventType NotificationEventType) ([]PushSubscription, error) {
	var subscriptions []PushSubscription

	// Get all users who have this notification enabled (or no preference set)
	// This is a bit complex because we need to handle the default "enabled" state

	// First, get all users with subscriptions
	if err := db.Find(&subscriptions).Error; err != nil {
		return nil, err
	}

	// Filter based on preferences
	var filtered []PushSubscription
	for _, sub := range subscriptions {
		if IsUserSubscribedToEvent(db, sub.UserID, eventType) {
			filtered = append(filtered, sub)
		}
	}

	return filtered, nil
}

// LogNotification creates a log entry for a sent notification
func LogNotification(db *gorm.DB, userID uint, eventType NotificationEventType, title, body, url string, serverID *uint, success bool, errorMsg string, metadata map[string]interface{}) (uint, error) {
	var metadataJSON string
	if metadata != nil {
		bytes, err := json.Marshal(metadata)
		if err != nil {
			log.Printf("failed to marshal notification metadata: %v", err)
		} else {
			metadataJSON = string(bytes)
		}
	}

	log := NotificationLog{
		UserID:    userID,
		EventType: eventType,
		Title:     title,
		Body:      body,
		URL:       url,
		ServerID:  serverID,
		Metadata:  metadataJSON,
		Success:   success,
		ErrorMsg:  errorMsg,
		SentAt:    time.Now(),
	}

	if err := db.Create(&log).Error; err != nil {
		return 0, err
	}
	return log.ID, nil
}

// GetNotificationHistory retrieves notification history for a user
func GetNotificationHistory(db *gorm.DB, userID uint, limit int) ([]NotificationLog, error) {
	if limit <= 0 {
		limit = 50
	}

	var logs []NotificationLog
	err := db.Where("user_id = ? AND dismissed = ?", userID, false).
		Order("read ASC").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	return logs, err
}

// CleanupInvalidSubscriptions removes subscriptions that have failed repeatedly
func CleanupInvalidSubscriptions(db *gorm.DB, endpoint string) error {
	return db.Where("endpoint = ?", endpoint).Delete(&PushSubscription{}).Error
}

// GetSubscriptionStats returns statistics about subscriptions
func GetSubscriptionStats(db *gorm.DB) (map[string]interface{}, error) {
	var totalSubs int64
	if err := db.Model(&PushSubscription{}).Count(&totalSubs).Error; err != nil {
		return nil, err
	}

	var uniqueUsers int64
	if err := db.Model(&PushSubscription{}).Distinct("user_id").Count(&uniqueUsers).Error; err != nil {
		return nil, err
	}

	var totalSent int64
	if err := db.Model(&NotificationLog{}).Count(&totalSent).Error; err != nil {
		return nil, err
	}

	var successfulSent int64
	if err := db.Model(&NotificationLog{}).Where("success = ?", true).Count(&successfulSent).Error; err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_subscriptions": totalSubs,
		"unique_users":        uniqueUsers,
		"total_sent":          totalSent,
		"successful_sent":     successfulSent,
	}

	if totalSent > 0 {
		stats["success_rate"] = fmt.Sprintf("%.2f%%", float64(successfulSent)/float64(totalSent)*100)
	} else {
		stats["success_rate"] = "N/A"
	}

	return stats, nil
}
