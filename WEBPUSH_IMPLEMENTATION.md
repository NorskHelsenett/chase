# Web Push Notifications - Complete Implementation Guide

This project includes a **fully functional, portable web push notification system** that can be easily copied to other projects.

## 📁 Package Structure

### Backend (Go)
```
api/webpush/
├── README.md           # Backend documentation
├── models.go           # Database models
├── vapid.go            # VAPID key generation and management
├── subscription.go     # Subscription management
├── sender.go           # Low-level push protocol implementation
├── notifier.go         # High-level notification helpers
└── handler.go          # HTTP API handlers
```

### Frontend (SvelteKit)
```
web/src/lib/push/
├── README.md                        # Frontend documentation
├── pushClient.js                    # JavaScript utilities
└── PushNotificationSettings.svelte  # UI component

web/static/
└── service-worker.js               # Service worker for push events
```

## 🚀 Quick Start

The system is **already integrated** into this project! Here's what's been set up:

### 1. Backend Integration ✅

- **Database**: Auto-migrates 4 tables (`vapid_keys`, `push_subscriptions`, `notification_preferences`, `notification_logs`)
- **VAPID Keys**: Automatically generated on first run
- **API Routes**: Available at `/api/push/*`
- **Notifications**: Automatically sent for:
  - New server added
  - Server goes offline
  - Server comes online
  - Server deleted
  - High/critical security risks found

### 2. Frontend Integration (To Do)

To enable the UI, add the component to your settings page:

```svelte
<!-- In web/src/routes/(app)/settings/+page.svelte or similar -->
<script>
	import PushNotificationSettings from '$lib/push/PushNotificationSettings.svelte';
</script>

<h2>Notifications</h2>
<PushNotificationSettings />
```

## 📡 Available Notification Events

| Event | Description | Trigger |
|-------|-------------|---------|
| `server_added` | New server added to monitoring | When user adds a server |
| `server_offline` | Server went offline | Status changes from online → offline |
| `server_online` | Server came back online | Status changes from offline → online |
| `server_deleted` | Server removed | When user deletes a server |
| `scan_completed` | Security scan finished | After security scan completes |
| `high_risk_found` | High/critical risk detected | When scan finds high/critical issues |

Users can enable/disable each event type individually in their preferences.

## 🔧 API Endpoints

All endpoints require authentication (except `/vapid-public-key`):

### Public
- `GET /api/push/vapid-public-key` - Get VAPID public key

### User Endpoints
- `POST /api/push/subscribe` - Subscribe to notifications
- `DELETE /api/push/unsubscribe` - Unsubscribe
- `GET /api/push/subscriptions` - List subscriptions
- `GET /api/push/preferences` - Get notification preferences
- `PUT /api/push/preferences` - Update preferences
- `GET /api/push/event-types` - Get event type metadata
- `GET /api/push/history` - Get notification history
- `POST /api/push/test` - Send test notification

### Admin Endpoints
- `GET /api/push/stats` - Get notification statistics

## 🎨 Testing

### 1. Test the Backend

```bash
# Start the application
go run api/main.go

# Check VAPID keys were generated
curl http://localhost:8080/api/push/vapid-public-key

# Get notification stats
curl http://localhost:8080/api/push/stats
```

### 2. Test from Browser

1. Open your application in a browser
2. Navigate to the settings page (with the component added)
3. Click "Enable Notifications"
4. Grant permission when prompted
5. Click "Send Test" to verify it works

### 3. Test Automated Notifications

```bash
# Add a server (triggers "server added" notification)
curl -X POST http://localhost:8080/api/servers \
  -H "Content-Type: application/json" \
  -d '{"url": "example.com", "active": true}'

# The monitoring system will automatically send notifications when:
# - Server goes offline
# - Server comes back online
# - Security scans find high risks
```

## 📦 Copying to Other Projects

### Option 1: Direct Copy (Recommended)

1. **Copy Backend Files**
```bash
# Copy the entire webpush package
cp -r api/webpush /your-project/api/

# Update import paths in copied files from:
# "github.com/norskhelsenett/chase/..."
# to:
# "github.com/yourorg/yourproject/..."
```

2. **Copy Frontend Files**
```bash
cp -r web/src/lib/push /your-project/src/lib/
cp web/static/service-worker.js /your-project/static/
```

3. **Initialize in Your App**
```go
// In your main.go or database init
import "github.com/yourorg/yourproject/webpush"

db := database.GetDB()
if err := webpush.InitDatabase(db); err != nil {
    log.Fatal(err)
}

// Register routes
pushHandler := webpush.NewHandler(db)
pushHandler.RegisterRoutes(apiRouter)
```

4. **Send Notifications**
```go
sender, _ := webpush.NewNotificationSender(db)

// Built-in helpers
sender.NotifyServerOffline("example.com", "Example Server")

// Or custom notifications
notification := &webpush.Notification{
    Title: "Custom Event",
    Body:  "Something happened!",
    Icon:  "/icon.png",
}
sender.SendToAll(webpush.EventServerAdded, notification)
```

### Option 2: Go Module (For Shared Usage)

```bash
# In your project
go get github.com/norskhelsenett/chase/webpush
```

Then import and use as shown above.

## 🔐 Security Considerations

- **HTTPS Required**: Service workers only work over HTTPS (or localhost)
- **VAPID Keys**: Generated once and stored in database - keep DB secure!
- **Subscriptions**: Include authentication tokens - handle with care
- **Permissions**: Users must grant permission - respect their choice
- **Privacy**: Notification content is encrypted end-to-end

## 🌐 Browser Compatibility

- Chrome/Edge 42+
- Firefox 44+
- Safari 16+ (macOS 13+, iOS 16.4+)
- Opera 29+

## 🛠️ Customization

### Add Custom Event Types

```go
// In your code
const (
    EventPaymentReceived NotificationEventType = "payment_received"
)

// Send notification
sender.SendToAll(EventPaymentReceived, &Notification{
    Title: "Payment Received",
    Body:  "$100 received",
})
```

### Customize Notification Appearance

```go
notification := &webpush.Notification{
    Title:   "Important Alert",
    Body:    "Something needs attention",
    Icon:    "/custom-icon.png",
    Badge:   "/badge.png",
    Image:   "/large-image.png",
    URL:     "/specific-page",
    Tag:     "unique-tag",  // Replaces previous notifications with same tag
    Data: map[string]interface{}{
        "customField": "value",
    },
    Actions: []webpush.NotificationAction{
        {Action: "view", Title: "View Now"},
        {Action: "dismiss", Title: "Dismiss"},
    },
}
```

### Adjust Service Worker Behavior

Edit `web/static/service-worker.js`:

```javascript
// Change notification defaults
let notification = {
    title: 'Your App Name',
    icon: '/your-icon.png',
    // ... customize
};

// Add custom click handling
self.addEventListener('notificationclick', (event) => {
    const data = event.notification.data;
    
    // Your custom logic
    if (data.type === 'custom_event') {
        clients.openWindow('/custom-page');
    }
});
```

## 📊 Monitoring & Stats

View notification statistics:

```bash
curl http://localhost:8080/api/push/stats
```

Returns:
```json
{
  "stats": {
    "total_subscriptions": 25,
    "unique_users": 12,
    "total_sent": 450,
    "successful_sent": 442,
    "success_rate": "98.22%"
  }
}
```

## 🐛 Troubleshooting

### Notifications not appearing?

1. Check browser console for errors
2. Verify HTTPS is enabled (or using localhost)
3. Confirm notification permission is granted
4. Check service worker is registered:
   ```javascript
   navigator.serviceWorker.getRegistration().then(console.log)
   ```
5. Review `/api/push/history` for error messages

### Subscriptions failing?

1. Ensure VAPID keys are generated (check logs)
2. Verify user is authenticated
3. Check browser compatibility
4. Review backend logs for errors

### Status changes not triggering notifications?

1. Verify monitoring is running
2. Check server is active
3. Review logs for notification errors
4. Ensure user has enabled the event type in preferences

## 📝 Additional Resources

- Backend documentation: `api/webpush/README.md`
- Frontend documentation: `web/src/lib/push/README.md`
- Web Push Protocol: https://developers.google.com/web/fundamentals/push-notifications
- VAPID Spec: https://datatracker.ietf.org/doc/html/rfc8292

## 🤝 Contributing

This implementation is designed to be portable and reusable. If you make improvements, consider:

1. Keeping it self-contained (minimal dependencies)
2. Documenting new features
3. Maintaining backward compatibility
4. Adding tests

## 📄 License

See project root for license information.

---

**Need help?** Check the detailed README files in each package directory or review the inline code comments.
