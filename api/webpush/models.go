package webpush

import (
	"time"

	"gorm.io/gorm"
)

// VAPIDKeys stores the VAPID key pair for web push notifications
// These keys are generated once and reused for all push notifications
type VAPIDKeys struct {
	ID         uint   `gorm:"primaryKey"`
	PublicKey  string `gorm:"type:text;not null"`
	PrivateKey string `gorm:"type:text;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PushSubscription stores a user's push notification subscription
type PushSubscription struct {
	gorm.Model
	UserID   uint   `gorm:"index;not null"`
	Endpoint string `gorm:"type:text;not null;uniqueIndex"`
	Auth     string `gorm:"type:text;not null"` // base64 encoded auth key
	P256dh   string `gorm:"type:text;not null"` // base64 encoded p256dh key
}

// NotificationPreference stores which notification types a user wants to receive
type NotificationPreference struct {
	gorm.Model
	UserID    uint                  `gorm:"index;not null;uniqueIndex:idx_user_event"`
	EventType NotificationEventType `gorm:"type:varchar(50);not null;uniqueIndex:idx_user_event"`
	Enabled   bool                  `gorm:"default:true"`
}

// NotificationEventType represents the type of notification event
type NotificationEventType string

const (
	EventServerAdded       NotificationEventType = "server_added"
	EventServerOffline     NotificationEventType = "server_offline"
	EventServerOnline      NotificationEventType = "server_online"
	EventServerDeleted     NotificationEventType = "server_deleted"
	EventServerDeactivated NotificationEventType = "server_deactivated"
	EventScanCompleted     NotificationEventType = "scan_completed"
	EventHighRiskFound     NotificationEventType = "high_risk_found"
)

// NotificationLog stores a history of sent notifications
type NotificationLog struct {
	gorm.Model
	UserID    uint                  `gorm:"index" json:"user_id"`
	EventType NotificationEventType `gorm:"type:varchar(50);index" json:"event_type"`
	Title     string                `gorm:"type:varchar(255)" json:"title"`
	Body      string                `gorm:"type:text" json:"body"`
	URL       string                `gorm:"type:text" json:"url"`
	Success   bool                  `gorm:"default:false" json:"sent"`
	ErrorMsg  string                `gorm:"type:text" json:"error_msg,omitempty"`
	SentAt    time.Time             `json:"sent_at"`
	CreatedAt time.Time             `json:"created_at"`
}

// TableName overrides for consistent naming
func (VAPIDKeys) TableName() string {
	return "vapid_keys"
}

func (PushSubscription) TableName() string {
	return "push_subscriptions"
}

func (NotificationPreference) TableName() string {
	return "notification_preferences"
}

func (NotificationLog) TableName() string {
	return "notification_logs"
}

// GetAllEventTypes returns all available notification event types
func GetAllEventTypes() []NotificationEventType {
	return []NotificationEventType{
		EventServerAdded,
		EventServerOffline,
		EventServerOnline,
		EventServerDeleted,
		EventServerDeactivated,
		EventScanCompleted,
		EventHighRiskFound,
	}
}

// EventTypeMetadata provides human-readable information about event types
type EventTypeMetadata struct {
	Type        NotificationEventType `json:"type"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Icon        string                `json:"icon"`
}

// GetEventMetadata returns metadata for all event types
func GetEventMetadata() []EventTypeMetadata {
	return []EventTypeMetadata{
		{
			Type:        EventServerAdded,
			Name:        "New Server Added",
			Description: "Notify when a new server is added to monitoring",
			Icon:        "➕",
		},
		{
			Type:        EventServerOffline,
			Name:        "Server Offline",
			Description: "Notify when a monitored server goes offline",
			Icon:        "🔴",
		},
		{
			Type:        EventServerOnline,
			Name:        "Server Online",
			Description: "Notify when a server comes back online",
			Icon:        "🟢",
		},
		{
			Type:        EventServerDeleted,
			Name:        "Server Deleted",
			Description: "Notify when a server is removed from monitoring",
			Icon:        "🗑️",
		},
		{
			Type:        EventServerDeactivated,
			Name:        "Server Deactivated",
			Description: "Notify when a server is automatically deactivated due to failures",
			Icon:        "⏸️",
		},
		{
			Type:        EventScanCompleted,
			Name:        "Security Scan Completed",
			Description: "Notify when a security scan completes",
			Icon:        "🔍",
		},
		{
			Type:        EventHighRiskFound,
			Name:        "High Risk Found",
			Description: "Notify when a high or critical security risk is detected",
			Icon:        "⚠️",
		},
	}
}
