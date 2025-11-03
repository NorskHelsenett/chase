# Quick Start Example - Using Web Push in Your Code

## Example 1: Send a Simple Notification

```go
package main

import (
    "github.com/norskhelsenett/chase/database"
    "github.com/norskhelsenett/chase/webpush"
)

func notifyUserAboutEvent(userEmail string) {
    db := database.GetDB()
    
    // Create notification sender
    sender, err := webpush.NewNotificationSender(db)
    if err != nil {
        log.Printf("Failed to create sender: %v", err)
        return
    }
    
    // Create notification
    notification := &webpush.Notification{
        Title: "Something Happened!",
        Body:  "Check out this important event",
        Icon:  "/icon-192.png",
        URL:   "/events/123",
        Data: map[string]interface{}{
            "eventId": "123",
            "type":    "custom",
        },
    }
    
    // Get user ID from email
    var user types.User
    db.Where("email = ?", userEmail).First(&user)
    
    // Send to user
    err = sender.SendToUser(user.ID, notification)
    if err != nil {
        log.Printf("Failed to send: %v", err)
    }
}
```

## Example 2: Send to All Users Interested in an Event

```go
func notifyEveryoneAboutMaintenance() {
    db := database.GetDB()
    sender, _ := webpush.NewNotificationSender(db)
    
    notification := &webpush.Notification{
        Title: "Scheduled Maintenance",
        Body:  "System will be down for 30 minutes at 2 AM",
        Icon:  "/maintenance-icon.png",
        Badge: "/badge-warning.png",
        Tag:   "maintenance", // Replace previous maintenance notifications
    }
    
    // Send to all users who have this event enabled
    count, err := sender.SendToAll(webpush.EventServerAdded, notification)
    log.Printf("Sent to %d users", count)
}
```

## Example 3: Custom Notification with Action Buttons

```go
func notifyApprovalNeeded(requestID string) {
    db := database.GetDB()
    sender, _ := webpush.NewNotificationSender(db)
    
    notification := &webpush.Notification{
        Title: "Approval Required",
        Body:  "A new request needs your approval",
        Icon:  "/approval-icon.png",
        URL:   "/approvals/" + requestID,
        Tag:   "approval-" + requestID,
        Actions: []webpush.NotificationAction{
            {
                Action: "approve",
                Title:  "✅ Approve",
            },
            {
                Action: "reject",
                Title:  "❌ Reject",
            },
            {
                Action: "view",
                Title:  "👁 View Details",
            },
        },
        Data: map[string]interface{}{
            "requestId": requestID,
            "type":      "approval",
        },
    }
    
    sender.SendToAll(webpush.EventCustomType, notification)
}
```

## Example 4: Integration with Existing Code

```go
// When creating a new resource
func CreateResource(c *gin.Context) {
    // ... your existing code ...
    
    // After successfully creating the resource
    if err := db.Create(&resource).Error; err == nil {
        // Send notification asynchronously (non-blocking)
        go func() {
            sender, _ := webpush.NewNotificationSender(db)
            notification := &webpush.Notification{
                Title: "New Resource Created",
                Body:  fmt.Sprintf("Resource %s was created", resource.Name),
                URL:   "/resources/" + resource.ID,
            }
            sender.SendToAll(webpush.EventServerAdded, notification)
        }()
    }
    
    c.JSON(201, resource)
}
```

## Example 5: Conditional Notifications

```go
func notifyIfUrgent(severity string, message string) {
    if severity != "high" && severity != "critical" {
        return // Don't notify for low/medium
    }
    
    db := database.GetDB()
    sender, _ := webpush.NewNotificationSender(db)
    
    notification := &webpush.Notification{
        Title:   fmt.Sprintf("%s Alert", strings.ToUpper(severity)),
        Body:    message,
        Icon:    "/alert-icon.png",
        Badge:   "/badge-error.png",
        Urgency: "high", // Browser will show more prominently
    }
    
    sender.SendToAll(webpush.EventHighRiskFound, notification)
}
```

## Example 6: Add Custom Event Type

```go
// 1. Define your custom event type
package mypackage

import "github.com/norskhelsenett/chase/webpush"

const (
    EventPaymentReceived webpush.NotificationEventType = "payment_received"
    EventOrderShipped    webpush.NotificationEventType = "order_shipped"
    EventCommentAdded    webpush.NotificationEventType = "comment_added"
)

// 2. Use it
func NotifyPayment(amount float64) {
    sender, _ := webpush.NewNotificationSender(db)
    
    notification := &webpush.Notification{
        Title: "Payment Received",
        Body:  fmt.Sprintf("You received $%.2f", amount),
        Icon:  "/money-icon.png",
        URL:   "/payments",
    }
    
    sender.SendToAll(EventPaymentReceived, notification)
}
```

## Example 7: Frontend - Subscribe User

```javascript
// In your Svelte/JavaScript code
import { subscribeToPush, isPushSubscribed } from '$lib/push/pushClient.js';

async function handleEnableNotifications() {
    try {
        // Check if already subscribed
        const subscribed = await isPushSubscribed();
        
        if (!subscribed) {
            // Subscribe the user
            await subscribeToPush();
            console.log('Successfully subscribed!');
        }
    } catch (error) {
        console.error('Failed to subscribe:', error);
    }
}
```

## Example 8: Frontend - Update Preferences

```javascript
import { 
    getNotificationPreferences, 
    updateNotificationPreferences 
} from '$lib/push/pushClient.js';

async function disableOfflineNotifications() {
    // Get current preferences
    const prefs = await getNotificationPreferences();
    
    // Update specific preference
    prefs.server_offline = false;
    prefs.server_online = false;
    
    // Save to server
    await updateNotificationPreferences(prefs);
}
```

## Example 9: Testing Your Notifications

```go
// Create a test endpoint
func TestNotificationHandler(c *gin.Context) {
    email := c.Query("email")
    
    db := database.GetDB()
    sender, err := webpush.NewNotificationSender(db)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    var user types.User
    if err := db.Where("email = ?", email).First(&user).Error; err != nil {
        c.JSON(404, gin.H{"error": "User not found"})
        return
    }
    
    notification := &webpush.Notification{
        Title: "Test Notification",
        Body:  "Testing 1, 2, 3...",
        Icon:  "/icon.png",
    }
    
    if err := sender.SendToUser(user.ID, notification); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Notification sent!"})
}

// Usage: curl "http://localhost:8080/test-notify?email=user@example.com"
```

## Example 10: Notification with Images

```go
func notifyWithScreenshot(serverURL, screenshotURL string) {
    sender, _ := webpush.NewNotificationSender(db)
    
    notification := &webpush.Notification{
        Title: "Server Screenshot Captured",
        Body:  fmt.Sprintf("New screenshot available for %s", serverURL),
        Icon:  "/camera-icon.png",
        Image: screenshotURL, // Large image shown in notification
        URL:   "/screenshots/" + serverURL,
        Tag:   "screenshot-" + serverURL,
    }
    
    sender.SendToAll(webpush.EventScanCompleted, notification)
}
```

## Pro Tips

### 1. Always use goroutines for notifications (non-blocking)
```go
// ✅ GOOD - Non-blocking
go notifyUsers(event)

// ❌ BAD - Blocks your request
notifyUsers(event)
```

### 2. Add meaningful data for client-side handling
```go
notification := &webpush.Notification{
    Data: map[string]interface{}{
        "serverId":   server.ID,
        "severity":   "high",
        "actionUrl":  "/servers/" + server.ID,
        "timestamp":  time.Now().Unix(),
    },
}
```

### 3. Use tags to replace notifications
```go
// Setting the same tag will replace the previous notification
notification.Tag = "server-status-" + serverID
```

### 4. Handle errors gracefully
```go
sender, err := webpush.NewNotificationSender(db)
if err != nil {
    log.Printf("Notification sender error: %v", err)
    return // Don't let notification failures break your main flow
}
```

### 5. Test with the built-in test endpoint
```bash
# After subscribing in browser
curl -X POST http://localhost:8080/api/push/test \
  -H "Cookie: session=YOUR_SESSION"
```

## Common Patterns

### Pattern 1: Notify on Status Change
```go
func UpdateStatus(oldStatus, newStatus string) {
    if oldStatus != newStatus {
        go notifyStatusChange(oldStatus, newStatus)
    }
}
```

### Pattern 2: Notify with Retry
```go
func notifyWithRetry(notification *webpush.Notification, retries int) {
    for i := 0; i < retries; i++ {
        err := sender.SendToAll(eventType, notification)
        if err == nil {
            return
        }
        time.Sleep(time.Second * time.Duration(i+1))
    }
}
```

### Pattern 3: Batch Notifications
```go
func notifyBatch(events []Event) {
    sender, _ := webpush.NewNotificationSender(db)
    
    for _, event := range events {
        notification := createNotificationFromEvent(event)
        go sender.SendToAll(event.Type, notification)
    }
}
```

That's it! You're ready to use web push notifications in your application. 🚀
