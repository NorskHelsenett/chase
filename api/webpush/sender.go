package webpush

import (
	"encoding/json"
	"fmt"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// Notification represents a web push notification payload
type Notification struct {
	Title   string                 `json:"title"`
	Body    string                 `json:"body"`
	Icon    string                 `json:"icon,omitempty"`
	Badge   string                 `json:"badge,omitempty"`
	Image   string                 `json:"image,omitempty"`
	URL     string                 `json:"url,omitempty"`
	Tag     string                 `json:"tag,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Actions []NotificationAction   `json:"actions,omitempty"`
}

// NotificationAction represents an action button on the notification
type NotificationAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// SendOptions contains options for sending a push notification
type SendOptions struct {
	TTL             int    // Time to live in seconds (default: 2419200 = 4 weeks)
	Urgency         string // low, normal, high, very-low
	Topic           string // For replacing notifications
	VAPIDPublicKey  string
	VAPIDPrivateKey string
}

// SendNotification sends a push notification to a specific subscription
func SendNotification(subscription *PushSubscription, notification *Notification, options *SendOptions) error {
	// Marshal notification to JSON
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %v", err)
	}

	// Create webpush subscription
	sub := &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			Auth:   subscription.Auth,
			P256dh: subscription.P256dh,
		},
	}

	// Set options
	ttl := 2419200 // 4 weeks default
	if options != nil && options.TTL > 0 {
		ttl = options.TTL
	}

	vapidPublicKey := ""
	vapidPrivateKey := ""
	subscriber := "mailto:noreply@example.com"

	if options != nil {
		vapidPublicKey = options.VAPIDPublicKey
		vapidPrivateKey = options.VAPIDPrivateKey
		if options.Topic != "" {
			subscriber = options.Topic
		}
	}

	// Validate VAPID keys are present
	if vapidPublicKey == "" || vapidPrivateKey == "" {
		return fmt.Errorf("VAPID keys are required for sending notifications")
	}

	// Log key lengths for debugging (don't log actual keys!)
	log.Printf("[WebPush] Sending notification with VAPID public key length: %d, private key length: %d",
		len(vapidPublicKey), len(vapidPrivateKey))

	// Send the notification using webpush-go
	resp, err := webpush.SendNotification(payload, sub, &webpush.Options{
		Subscriber:      subscriber,
		VAPIDPublicKey:  vapidPublicKey,
		VAPIDPrivateKey: vapidPrivateKey,
		TTL:             ttl,
		Urgency:         webpush.UrgencyNormal,
	})
	if err != nil {
		return fmt.Errorf("failed to send notification (check VAPID key format): %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == 201 || resp.StatusCode == 200 {
		return nil // Success
	}

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return fmt.Errorf("subscription expired (status %d)", resp.StatusCode)
	}

	return fmt.Errorf("push service error: status %d", resp.StatusCode)
}
