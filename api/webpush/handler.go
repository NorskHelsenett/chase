package webpush

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/types"
	"gorm.io/gorm"
)

// Handler manages web push HTTP endpoints
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new web push handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// getUserIDByEmail gets or creates a user ID from email
func (h *Handler) getUserIDByEmail(email string) (uint, error) {
	var user types.User
	err := h.db.Where("email = ?", email).First(&user).Error

	if err == nil {
		return user.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// Create new user if doesn't exist
	user = types.User{
		Email: email,
		Name:  email, // Default name to email
	}

	if err := h.db.Create(&user).Error; err != nil {
		return 0, err
	}

	return user.ID, nil
}

// RegisterRoutes registers all web push routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	push := router.Group("/push")
	{
		// Public endpoint to get VAPID public key
		push.GET("/vapid-public-key", h.GetVAPIDPublicKey)

		// Subscription management (requires authentication)
		push.POST("/subscribe", h.Subscribe)
		push.DELETE("/unsubscribe", h.Unsubscribe)
		push.GET("/subscriptions", h.GetSubscriptions)

		// Notification preferences
		push.GET("/preferences", h.GetPreferences)
		push.PUT("/preferences", h.UpdatePreferences)

		// Event types metadata
		push.GET("/event-types", h.GetEventTypes)

		// Notification history
		push.GET("/history", h.GetHistory)

		// Stats (admin only - you may want to add admin middleware)
		push.GET("/stats", h.GetStats)

		// Test notification (for development/testing)
		push.POST("/test", h.SendTestNotification)

		// Admin endpoints for VAPID key management
		push.GET("/admin/vapid-keys-status", h.GetVAPIDKeysStatus)
		push.POST("/admin/regenerate-vapid-keys", h.RegenerateVAPIDKeysHandler)
	}
}

// GetVAPIDPublicKey returns the public VAPID key for client-side use
func (h *Handler) GetVAPIDPublicKey(c *gin.Context) {
	keys, err := GetVAPIDKeys(h.db)
	if err != nil {
		log.Printf("Failed to get VAPID keys: %v", err)
		c.JSON(500, gin.H{"error": "Failed to get VAPID public key"})
		return
	}

	// Validate the keys are in correct format
	_, _, valErr := normalizeVAPIDKeyPair(keys.PublicKey, keys.PrivateKey)
	if valErr != nil {
		log.Printf("VAPID keys are invalid: %v. They may need to be regenerated.", valErr)
		c.JSON(500, gin.H{
			"error":   "VAPID keys are invalid",
			"details": "Please contact administrator to regenerate VAPID keys",
		})
		return
	}

	c.JSON(200, gin.H{"publicKey": keys.PublicKey})
}

// Subscribe handles push subscription requests
func (h *Handler) Subscribe(c *gin.Context) {
	// Get email from context (set by auth middleware)
	email, exists := c.Get("email")
	if !exists {
		c.JSON(401, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
		Keys     struct {
			Auth   string `json:"auth" binding:"required"`
			P256dh string `json:"p256dh" binding:"required"`
		} `json:"keys" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// Get or create user
	userID, err := h.getUserIDByEmail(email.(string))
	if err != nil {
		log.Printf("Failed to get user ID: %v", err)
		c.JSON(500, gin.H{"error": "Failed to get user information"})
		return
	}

	err = SubscribeUser(h.db, userID, req.Endpoint, req.Keys.Auth, req.Keys.P256dh)
	if err != nil {
		log.Printf("Failed to subscribe user %d: %v", userID, err)
		c.JSON(500, gin.H{"error": "Failed to save subscription"})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "Subscription saved"})
}

// Unsubscribe handles unsubscription requests
func (h *Handler) Unsubscribe(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(401, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	userID, err := h.getUserIDByEmail(email.(string))
	if err != nil {
		log.Printf("Failed to get user ID: %v", err)
		c.JSON(500, gin.H{"error": "Failed to get user information"})
		return
	}

	if req.Endpoint != "" {
		err = UnsubscribeUser(h.db, userID, req.Endpoint)
	} else {
		err = UnsubscribeAllForUser(h.db, userID)
	}

	if err != nil {
		log.Printf("Failed to unsubscribe user %d: %v", userID, err)
		c.JSON(500, gin.H{"error": "Failed to remove subscription"})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "Unsubscribed"})
}

// GetSubscriptions returns all subscriptions for the current user
func (h *Handler) GetSubscriptions(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(401, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := h.getUserIDByEmail(email.(string))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get user information"})
		return
	}

	subscriptions, err := GetUserSubscriptions(h.db, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get subscriptions"})
		return
	}

	// Return limited info (don't expose auth keys)
	result := make([]map[string]interface{}, len(subscriptions))
	for i, sub := range subscriptions {
		result[i] = map[string]interface{}{
			"id":       sub.ID,
			"endpoint": sub.Endpoint,
			"created":  sub.CreatedAt,
		}
	}

	c.JSON(200, gin.H{"subscriptions": result})
}

// GetPreferences returns notification preferences for the current user
func (h *Handler) GetPreferences(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(401, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := h.getUserIDByEmail(email.(string))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get user information"})
		return
	}

	prefs, err := GetNotificationPreferences(h.db, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get preferences"})
		return
	}

	c.JSON(200, gin.H{"preferences": prefs})
}

// UpdatePreferences updates notification preferences
func (h *Handler) UpdatePreferences(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(401, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := h.getUserIDByEmail(email.(string))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get user information"})
		return
	}

	var req map[string]bool
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// Update each preference
	for eventTypeStr, enabled := range req {
		eventType := NotificationEventType(eventTypeStr)
		if err := SetNotificationPreference(h.db, userID, eventType, enabled); err != nil {
			log.Printf("Failed to set preference for %s: %v", eventType, err)
			c.JSON(500, gin.H{"error": "Failed to update preferences"})
			return
		}
	}

	c.JSON(200, gin.H{"success": true, "message": "Preferences updated"})
}

// GetEventTypes returns metadata about available notification event types
func (h *Handler) GetEventTypes(c *gin.Context) {
	metadata := GetEventMetadata()
	c.JSON(200, gin.H{"eventTypes": metadata})
}

// GetHistory returns notification history for the current user
func (h *Handler) GetHistory(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(401, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := h.getUserIDByEmail(email.(string))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get user information"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	history, err := GetNotificationHistory(h.db, userID, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get history"})
		return
	}

	c.JSON(200, gin.H{"history": history})
}

// GetStats returns statistics about push notifications
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := GetSubscriptionStats(h.db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(200, gin.H{"stats": stats})
}

// SendTestNotification sends a test notification to the current user
func (h *Handler) SendTestNotification(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(401, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := h.getUserIDByEmail(email.(string))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get user information"})
		return
	}

	sender, err := NewNotificationSender(h.db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to initialize notification sender"})
		return
	}

	notification := &Notification{
		Title: "Test Notification",
		Body:  "This is a test notification from Chase",
		Icon:  "/icon-192.png",
		Data: map[string]interface{}{
			"test": true,
			"url":  "/",
		},
	}

	err = sender.SendToUser(userID, notification)
	if err != nil {
		log.Printf("Failed to send test notification: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to send notification: %v", err)})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "Test notification sent"})
}

// Middleware to extract user ID from context
// This should be adapted to your authentication system
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement your authentication logic here
		// For now, this is a placeholder
		// You should extract the user ID from your session/JWT/etc

		// Example: Get from custom header (replace with your auth logic)
		userIDHeader := c.GetHeader("X-User-ID")
		if userIDHeader != "" {
			userID, err := strconv.ParseUint(userIDHeader, 10, 32)
			if err == nil {
				c.Set("userID", uint(userID))
			}
		}

		c.Next()
	}
}

// GetVAPIDKeysStatus checks the validity of current VAPID keys
func (h *Handler) GetVAPIDKeysStatus(c *gin.Context) {
	// TODO: Add admin authentication check here

	keys, err := GetVAPIDKeys(h.db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get VAPID keys", "details": err.Error()})
		return
	}

	// Validate the keys
	normalizedPub, normalizedPriv, valErr := normalizeVAPIDKeyPair(keys.PublicKey, keys.PrivateKey)

	status := gin.H{
		"publicKeyLength":  len(keys.PublicKey),
		"privateKeyLength": len(keys.PrivateKey),
		"publicKeyPreview": keys.PublicKey[:20] + "...",
	}

	if valErr != nil {
		status["valid"] = false
		status["error"] = valErr.Error()
		status["recommendation"] = "Keys are invalid and should be regenerated"
		c.JSON(200, status)
		return
	}

	status["valid"] = true
	status["normalized"] = (normalizedPub == keys.PublicKey && normalizedPriv == keys.PrivateKey)

	c.JSON(200, status)
}

// RegenerateVAPIDKeysHandler regenerates VAPID keys (admin only)
func (h *Handler) RegenerateVAPIDKeysHandler(c *gin.Context) {
	// TODO: Add admin authentication check here

	log.Println("Admin requested VAPID key regeneration")

	if err := RegenerateVAPIDKeys(h.db); err != nil {
		log.Printf("Failed to regenerate VAPID keys: %v", err)
		c.JSON(500, gin.H{"error": "Failed to regenerate VAPID keys", "details": err.Error()})
		return
	}

	keys, err := GetVAPIDKeys(h.db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to retrieve new keys", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"success":          true,
		"message":          "VAPID keys regenerated successfully. All subscriptions have been cleared.",
		"publicKey":        keys.PublicKey,
		"publicKeyLength":  len(keys.PublicKey),
		"privateKeyLength": len(keys.PrivateKey),
	})
}
