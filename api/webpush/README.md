# Web Push Notifications Package

A complete, portable web push notification system for Go (Gin) backends with SvelteKit frontends. This package implements the full Web Push Protocol with VAPID authentication.

## Features

- ✅ Automatic VAPID key generation and storage
- ✅ User subscription management
- ✅ Event-based notification preferences
- ✅ Notification history logging
- ✅ Built-in notification types for common events
- ✅ Easy integration with existing projects
- ✅ Production-ready encryption (aes128gcm)
- ✅ Automatic cleanup of expired subscriptions

## Quick Start

### Backend Integration (Go/Gin)

#### 1. Initialize the database

```go
import (
    "github.com/norskhelsenett/chase/webpush"
    "gorm.io/gorm"
)

// In your database initialization
func InitDatabase() error {
    db := database.GetDB()
    
    // Initialize webpush tables and VAPID keys
    if err := webpush.InitDatabase(db); err != nil {
        return err
    }
    
    return nil
}
```

#### 2. Register routes

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/norskhelsenett/chase/webpush"
)

func SetupRoutes(router *gin.Engine) {
    db := database.GetDB()
    
    // Create webpush handler
    pushHandler := webpush.NewHandler(db)
    
    // Register routes under /api
    api := router.Group("/api")
    pushHandler.RegisterRoutes(api)
}
```

#### 3. Send notifications

```go
import "github.com/norskhelsenett/chase/webpush"

// Send a notification when a server goes offline
func NotifyServerStatus(db *gorm.DB, serverURL, serverName string, isOnline bool) {
    sender, err := webpush.NewNotificationSender(db)
    if err != nil {
        log.Printf("Failed to create notification sender: %v", err)
        return
    }
    
    if isOnline {
        sender.NotifyServerOnline(serverURL, serverName)
    } else {
        sender.NotifyServerOffline(serverURL, serverName)
    }
}

// Or send custom notifications
func SendCustomNotification(db *gorm.DB, eventType webpush.NotificationEventType) {
    sender, _ := webpush.NewNotificationSender(db)
    
    notification := &webpush.Notification{
        Title: "Custom Notification",
        Body:  "Something important happened!",
        Icon:  "/icon.png",
        URL:   "/dashboard",
        Data: map[string]interface{}{
            "customField": "value",
        },
    }
    
    sender.SendToAll(eventType, notification)
}
```

### Frontend Integration (SvelteKit)

See the `web/` directory for complete SvelteKit components.

## API Endpoints

All endpoints are under `/api/push`:

### Public Endpoints

- `GET /vapid-public-key` - Get the VAPID public key for subscriptions

### Authenticated Endpoints

- `POST /subscribe` - Subscribe to push notifications
- `DELETE /unsubscribe` - Unsubscribe from notifications
- `GET /subscriptions` - List user's subscriptions
- `GET /preferences` - Get notification preferences
- `PUT /preferences` - Update notification preferences
- `GET /event-types` - Get available notification event types
- `GET /history` - Get notification history
- `POST /test` - Send a test notification

### Admin Endpoints

- `GET /stats` - Get notification statistics

## Built-in Event Types

The package includes these predefined event types:

- `server_added` - New server added to monitoring
- `server_offline` - Server went offline
- `server_online` - Server came back online
- `server_deleted` - Server removed from monitoring
- `scan_completed` - Security scan completed
- `high_risk_found` - High/critical security risk detected

## Database Models

The package creates these tables:

- `vapid_keys` - Stores VAPID public/private key pair
- `push_subscriptions` - User push subscriptions
- `notification_preferences` - User notification settings
- `notification_logs` - History of sent notifications

## Copying to Other Projects

### Method 1: Direct Copy (Recommended for simplicity)

1. Copy the entire `api/webpush/` directory to your project
2. Copy the `web/src/lib/push/` directory (see web section below)
3. Update import paths to match your module name
4. Initialize the database and routes as shown above

### Method 2: Go Module (For shared usage)

```bash
# In your project
go get github.com/norskhelsenett/chase/webpush
```

Then import as:
```go
import "github.com/norskhelsenett/chase/webpush"
```

## Configuration

### Environment Variables

You can customize these settings:

- `VAPID_SUBJECT` - Email for VAPID claims (default: mailto:noreply@example.com)

### Customization

#### Add Custom Event Types

```go
// In your code
const (
    EventCustomType NotificationEventType = "custom_event"
)

// Send notification
sender.SendToAll(EventCustomType, notification)
```

#### Custom Notification Logic

```go
// Create your own sender wrapper
type MyNotificationService struct {
    sender *webpush.NotificationSender
}

func (s *MyNotificationService) NotifyPaymentReceived(amount float64) {
    notification := &webpush.Notification{
        Title: "Payment Received",
        Body:  fmt.Sprintf("You received $%.2f", amount),
        // ... customize as needed
    }
    s.sender.SendToAll(webpush.EventCustomType, notification)
}
```

## Security Considerations

1. **VAPID Keys**: Generated automatically and stored in the database. Keep your database secure!
2. **Subscriptions**: Include user authentication tokens - never expose raw subscription data
3. **Endpoint Validation**: The package validates subscription endpoints
4. **Encryption**: All payloads are encrypted using aes128gcm

## Testing

### Send a Test Notification

```bash
# Using curl (replace with your auth token)
curl -X POST http://localhost:8080/api/push/test \
  -H "X-User-ID: 1" \
  -H "Content-Type: application/json"
```

### Check Statistics

```bash
curl http://localhost:8080/api/push/stats
```

## Troubleshooting

### Notifications not received

1. Check browser console for service worker errors
2. Verify VAPID public key matches between backend and frontend
3. Check notification permissions in browser
4. Review `/api/push/history` for error messages

### Subscription failures

1. Ensure HTTPS is enabled (required for service workers)
2. Verify VAPID keys are properly generated
3. Check browser compatibility (modern browsers required)

### Database errors

1. Ensure migrations ran successfully
2. Check database permissions
3. Review GORM logs for details

## Browser Compatibility

- Chrome/Edge 42+
- Firefox 44+
- Safari 16+ (macOS 13+, iOS 16.4+)
- Opera 29+

## License

See project root for license information.

## Contributing

This package is designed to be portable. If you add improvements, consider submitting a PR!

## Credits

Built for the Chase server monitoring platform by Norsk Helsenett.
