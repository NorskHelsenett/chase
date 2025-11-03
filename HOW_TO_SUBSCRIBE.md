# 🔔 How to Subscribe to Push Notifications

## Step-by-Step Guide

### 1. Navigate to Settings
- Go to **Settings** page in your Chase app
- You should see a new "Push Notifications" section at the top

### 2. Enable Notifications
- Click the **"🔔 Enable Notifications"** button
- Your browser will ask for permission - click **"Allow"**

### 3. Choose Your Preferences
After subscribing, you'll see checkboxes for each notification type:

- ✅ **New Server Added** - Get notified when servers are added
- ✅ **Server Offline** - Alert when a server goes down
- ✅ **Server Online** - Know when a server comes back
- ✅ **Server Deleted** - Notification when servers are removed
- ✅ **Security Scan Completed** - Updates on security scans
- ✅ **High Risk Found** - Immediate alerts for critical security issues

Toggle any events you don't want to receive.

### 4. Test It!
- Click the **"🧪 Send Test"** button
- You should see a test notification appear!

## What Happens Next?

Once subscribed, you'll automatically receive notifications when:
- You add a new server
- Any server goes offline or comes back online
- A security scan finds high/critical risks
- Someone deletes a server

## Troubleshooting

### "Push notifications are not supported"
- Make sure you're using a modern browser (Chrome, Firefox, Safari 16+, Edge)
- Check that you're on HTTPS (or localhost for development)

### "Push notifications are blocked"
- Go to browser settings → Site Settings → Notifications
- Find your site and change permission from "Block" to "Allow"
- Refresh the page and try again

### Permission dialog doesn't appear
- Your browser may have auto-blocked notifications
- Check the address bar for a blocked notification icon 🔔
- Click it and select "Allow"

### Notifications not appearing
1. Check "Do Not Disturb" mode is off on your device
2. Verify notification preferences are enabled
3. Try the "Send Test" button
4. Check browser notification settings

## Browser-Specific Notes

### Chrome/Edge
- Notifications work great on desktop and Android
- Make sure Chrome is allowed to show notifications in system settings

### Firefox
- Works on desktop and Android
- Check Firefox notification preferences in Settings

### Safari (macOS 13+ / iOS 16.4+)
- Make sure "Allow Websites to Ask for Permission to Send Notifications" is enabled
- System Settings → Notifications → Safari → Allow Notifications

## Testing Notifications

### Manual Test
1. Go to Settings
2. Click "🧪 Send Test" button
3. You should see a notification within seconds

### Real Event Test
1. Add a test server from the dashboard
2. You should get a "New Server Added" notification
3. Watch as the monitoring system sends "Server Online/Offline" notifications

## Managing Subscriptions

### View Your Subscriptions
- The settings page shows all your active subscriptions
- You can have multiple devices subscribed (phone, laptop, etc.)

### Unsubscribe
- Click **"🔕 Disable Notifications"** to unsubscribe
- This removes the current device's subscription
- Other devices stay subscribed

### Update Preferences Anytime
- Just toggle the checkboxes on/off
- Changes take effect immediately
- No need to re-subscribe

## Notification Features

### Smart Notifications
- **Tagged by type** - Multiple notifications of the same type replace each other
- **Clickable** - Click to jump directly to the relevant page
- **Rich content** - Icons, images, and action buttons
- **Persistent** - Won't disappear until you interact with them (on some events)

### Notification Content
Each notification includes:
- **Title** - What happened
- **Body** - Details about the event
- **Icon** - Visual indicator
- **Click action** - Opens the relevant page

Example:
```
Title: "Server Offline"
Body: "example.com is offline"
Click → Opens /server/example.com
```

## Privacy & Security

- ✅ Notifications are encrypted end-to-end
- ✅ Only you can see your notifications
- ✅ You control which events to receive
- ✅ Unsubscribe anytime
- ✅ No tracking or analytics

## API Endpoints (For Developers)

If you want to interact programmatically:

```bash
# Get VAPID public key
curl http://localhost:8080/api/push/vapid-public-key

# Subscribe (requires authentication)
curl -X POST http://localhost:8080/api/push/subscribe \
  -H "Content-Type: application/json" \
  -d '{"endpoint": "...", "keys": {"auth": "...", "p256dh": "..."}}'

# Get preferences
curl http://localhost:8080/api/push/preferences

# Update preferences
curl -X PUT http://localhost:8080/api/push/preferences \
  -H "Content-Type: application/json" \
  -d '{"server_offline": true, "server_online": false}'

# Send test notification
curl -X POST http://localhost:8080/api/push/test
```

## Next Steps

1. ✅ Subscribe to notifications
2. ✅ Customize your preferences
3. ✅ Test with the "Send Test" button
4. ✅ Add a server and watch for notifications
5. ✅ Enjoy real-time updates!

---

**Questions?** Check the console (F12) for any error messages or review the comprehensive documentation in:
- `api/webpush/README.md` - Backend details
- `web/src/lib/push/README.md` - Frontend details
- `WEBPUSH_EXAMPLES.md` - Code examples
