# Web Push Notifications - System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER'S BROWSER                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  SvelteKit Frontend (Web App)                                │  │
│  │                                                               │  │
│  │  ┌─────────────────────────────────────────────────────┐    │  │
│  │  │ PushNotificationSettings.svelte                     │    │  │
│  │  │  • Enable/Disable notifications                     │    │  │
│  │  │  • Configure preferences                            │    │  │
│  │  │  • View history                                     │    │  │
│  │  └─────────────────────────────────────────────────────┘    │  │
│  │         │                                                    │  │
│  │         ↓                                                    │  │
│  │  ┌─────────────────────────────────────────────────────┐    │  │
│  │  │ pushClient.js                                       │    │  │
│  │  │  • subscribeToPush()                                │    │  │
│  │  │  • unsubscribeFromPush()                            │    │  │
│  │  │  • getNotificationPreferences()                     │    │  │
│  │  │  • updateNotificationPreferences()                  │    │  │
│  │  └─────────────────────────────────────────────────────┘    │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Service Worker (/static/service-worker.js)                 │  │
│  │                                                               │  │
│  │  • Receives push events from browser                         │  │
│  │  • Shows notifications                                        │  │
│  │  • Handles notification clicks                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                          ↑                                          │
│                          │ Push Messages                            │
│                          │ (encrypted)                              │
└──────────────────────────┼──────────────────────────────────────────┘
                           │
                           │
┌──────────────────────────┼──────────────────────────────────────────┐
│                          │ Push Service (Browser Vendor)             │
│                    ┌─────┴─────┐                                    │
│                    │  Chrome   │  Firefox  │  Safari  │  Edge       │
│                    │  FCM      │  Mozilla  │  APNs    │  WNS        │
│                    └─────┬─────┘                                    │
└──────────────────────────┼──────────────────────────────────────────┘
                           │
                           ↑ HTTPS POST
                           │ (encrypted payload + VAPID auth)
                           │
┌──────────────────────────┼──────────────────────────────────────────┐
│                    CHASE SERVER                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  HTTP API Routes (/api/push/*)                               │  │
│  │                                                               │  │
│  │  GET  /vapid-public-key      │  POST /subscribe              │  │
│  │  GET  /preferences            │  PUT  /preferences            │  │
│  │  GET  /event-types            │  GET  /history                │  │
│  │  POST /test                   │  DELETE /unsubscribe          │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                          │                                          │
│                          ↓                                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  webpush.Handler                                             │  │
│  │  • Subscribe()         • GetPreferences()                    │  │
│  │  • Unsubscribe()       • UpdatePreferences()                 │  │
│  │  • SendTestNotification()                                     │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                          │                                          │
│                          ↓                                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  webpush Package (Core Logic)                                │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │ subscription.go                                         │  │  │
│  │  │  • SubscribeUser()                                      │  │  │
│  │  │  • GetNotificationPreferences()                         │  │  │
│  │  │  • SetNotificationPreference()                          │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │ vapid.go                                                │  │  │
│  │  │  • InitDatabase() - auto-generates VAPID keys          │  │  │
│  │  │  • GetVAPIDKeys()                                       │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │ sender.go                                               │  │  │
│  │  │  • SendNotification() - implements Web Push Protocol   │  │  │
│  │  │  • encryptPayload() - aes128gcm encryption             │  │  │
│  │  │  • generateVAPIDAuthHeader() - JWT signing             │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │ notifier.go                                             │  │  │
│  │  │  • NotifyServerAdded()                                  │  │  │
│  │  │  • NotifyServerOffline()                                │  │  │
│  │  │  • NotifyServerOnline()                                 │  │  │
│  │  │  • NotifyHighRiskFound()                                │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                          │                                          │
│                          ↓                                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Application Integration Points                              │  │
│  │                                                               │  │
│  │  ┌────────────────────┐  ┌─────────────────────────────┐    │  │
│  │  │ servers/handler.go │  │ servers/database.go         │    │  │
│  │  │                    │  │                             │    │  │
│  │  │ AddServer()        │  │ runMonitoring()             │    │  │
│  │  │  └─> NotifyServer- │  │  └─> NotifyServerStatus-    │    │  │
│  │  │      Added()       │  │      Change()               │    │  │
│  │  │                    │  │                             │    │  │
│  │  │ DeleteServer()     │  │                             │    │  │
│  │  │  └─> NotifyServer- │  │                             │    │  │
│  │  │      Deleted()     │  │                             │    │  │
│  │  └────────────────────┘  └─────────────────────────────┘    │  │
│  │                                                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │ security/handler.go                                    │  │  │
│  │  │                                                         │  │  │
│  │  │ storeSecurityReport()                                  │  │  │
│  │  │  └─> notifyHighRisk() (if critical/high risk)         │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                          │                                          │
│                          ↓                                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  SQLite Database                                             │  │
│  │                                                               │  │
│  │  ┌─────────────────┐  ┌──────────────────────────────────┐  │  │
│  │  │ vapid_keys      │  │ push_subscriptions               │  │  │
│  │  │ • public_key    │  │ • user_id                        │  │  │
│  │  │ • private_key   │  │ • endpoint                       │  │  │
│  │  └─────────────────┘  │ • auth (encryption key)          │  │  │
│  │                       │ • p256dh (public key)            │  │  │
│  │  ┌─────────────────┐  └──────────────────────────────────┘  │  │
│  │  │ notification_   │  ┌──────────────────────────────────┐  │  │
│  │  │ preferences     │  │ notification_logs                │  │  │
│  │  │ • user_id       │  │ • user_id                        │  │  │
│  │  │ • event_type    │  │ • event_type                     │  │  │
│  │  │ • enabled       │  │ • title / body                   │  │  │
│  │  └─────────────────┘  │ • success / error_msg            │  │  │
│  │                       └──────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘


EVENT FLOW EXAMPLES:
═══════════════════

1. Server Goes Offline:
   runMonitoring() → detects status change → NotifyServerStatusChange()
   → NotificationSender.NotifyServerOffline() → SendNotification()
   → POST to Push Service → Browser receives → Service Worker shows notification

2. User Subscribes:
   Browser → /api/push/subscribe → Handler.Subscribe() → SubscribeUser()
   → Store in push_subscriptions table → Response with success

3. User Updates Preferences:
   Frontend → /api/push/preferences (PUT) → Handler.UpdatePreferences()
   → SetNotificationPreference() → Update notification_preferences table


NOTIFICATION EVENT TYPES:
═════════════════════════

✅ server_added      - New server added to monitoring
✅ server_offline    - Server went offline
✅ server_online     - Server came back online
✅ server_deleted    - Server removed from monitoring
✅ scan_completed    - Security scan completed
✅ high_risk_found   - High/critical security risk detected


KEY TECHNOLOGIES:
════════════════

• Web Push Protocol (RFC 8030)
• VAPID (RFC 8292)
• aes128gcm Encryption (RFC 8291)
• ECDH Key Exchange (P-256)
• HKDF Key Derivation
• ES256 JWT Signing
• Service Workers API
• Push API
• Notification API
```
