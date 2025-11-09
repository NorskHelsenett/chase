package webpush

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// HasNotificationSince checks whether a successful notification for the given server/event has been logged since the provided time.
func HasNotificationSince(db *gorm.DB, serverID uint, eventType NotificationEventType, since time.Time) (bool, error) {
	if db == nil || serverID == 0 {
		return false, nil
	}

	var record struct {
		ID uint
	}

	err := db.Table(NotificationLog{}.TableName()).
		Select("id").
		Where("server_id = ? AND event_type = ? AND success = ? AND created_at >= ?", serverID, eventType, true, since).
		Order("id DESC").
		Limit(1).
		Scan(&record).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return record.ID != 0, nil
}

// ShouldThrottleNotification returns true when a notification was already sent within the provided cooldown period.
func ShouldThrottleNotification(db *gorm.DB, serverID uint, eventType NotificationEventType, cooldown time.Duration) (bool, error) {
	if cooldown <= 0 {
		return false, nil
	}

	cutoff := time.Now().Add(-cooldown)
	return HasNotificationSince(db, serverID, eventType, cutoff)
}
