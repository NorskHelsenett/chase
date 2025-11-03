# Web Push Notifications - Troubleshooting Guide

## Notifications Not Appearing in Browser

If you've successfully subscribed but aren't seeing notifications, here are the most common causes and solutions:

### 1. Check Browser Console

Open your browser's Developer Tools (F12) and check the Console tab for errors:

```javascript
// Check if service worker is registered
navigator.serviceWorker.getRegistrations().then(regs => {
  console.log('Service workers:', regs);
});

// Check current subscription
navigator.serviceWorker.ready.then(reg => {
  reg.pushManager.getSubscription().then(sub => {
    console.log('Current subscription:', sub);
  });
});

// Check notification permission
console.log('Notification permission:', Notification.permission);
```

### 2. Localhost HTTPS Requirements

Web Push requires HTTPS, except for `localhost`. However, some browsers may still have issues:

**Solutions:**
- Use `http://localhost:5173` (not `http://127.0.0.1:5173`)
- Try a different browser (Chrome, Firefox, Edge all have different behaviors)
- Test in an incognito/private window
- For production, always use HTTPS

### 3. Service Worker Registration

Check if the service worker is properly registered:

```javascript
// In browser console
navigator.serviceWorker.getRegistrations().then(registrations => {
  console.log('Registered service workers:', registrations.length);
  registrations.forEach(reg => {
    console.log('Scope:', reg.scope);
    console.log('Active:', reg.active);
  });
});
```

**Common Issues:**
- Service worker file not at root (`/service-worker.js`)
- Caching issues - try hard refresh (Ctrl+Shift+R)
- Scope mismatch

**Fix:** Clear all service workers and re-register:
```javascript
navigator.serviceWorker.getRegistrations().then(registrations => {
  registrations.forEach(reg => reg.unregister());
  location.reload();
});
```

### 4. Notification Permission

The browser must have notification permission granted:

```javascript
// Check permission status
console.log('Permission:', Notification.permission);
// Should show: "granted"

// If "denied" or "default", need to reset:
// - Chrome: Settings → Privacy → Site Settings → Notifications
// - Firefox: Address bar → 🔒 icon → Permissions → Notifications
// - Edge: Settings → Cookies and site permissions → Notifications
```

### 5. Browser-Specific Issues

#### Chrome/Chromium
- Check `chrome://settings/content/notifications`
- Ensure site is not in the "Block" list
- Clear browsing data and try again

#### Firefox
- Check `about:preferences#privacy` → Permissions → Notifications
- Firefox may block notifications if tab is not visible
- Try enabling "Show a notification when websites ask for permission to send notifications"

#### Safari
- macOS Safari requires notification permission at system level
- Check System Preferences → Notifications
- Safari 16+ required for full Web Push support

### 6. Test Notification Manually

Send a test notification from the browser console:

```javascript
// Create a test notification (bypasses service worker)
if (Notification.permission === 'granted') {
  new Notification('Test Notification', {
    body: 'This is a test from the browser console',
    icon: '/favicon.png',
    badge: '/badge.png'
  });
}
```

If this works but server notifications don't, the issue is with the service worker or push subscription.

### 7. Check Network Requests

In DevTools → Network tab, verify:

1. **Subscription request** succeeds:
   - `POST /api/push/subscribe` → Status 200
   - Response contains success message

2. **Test notification request** succeeds:
   - `POST /api/push/test` → Status 200
   - Server logs show notification sent

3. **No CORS errors** in console

### 8. Service Worker Debugging

Check if service worker is receiving push events:

Edit `/static/service-worker.js` to add logging:

```javascript
self.addEventListener('push', function (event) {
  console.log('[Service Worker] Push event received:', event);
  console.log('[Service Worker] Push data:', event.data?.text());
  
  // ... rest of code
});
```

Then check Service Worker console:
- DevTools → Application → Service Workers → Click on service worker
- Should show a separate console for service worker logs

### 9. Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| "Push notifications are not supported" | Browser doesn't support Push API | Use Chrome, Firefox, or Edge |
| "Notification permission not granted" | User denied permission | Reset permission in browser settings |
| "Failed to subscribe" | Service worker not registered | Hard refresh page |
| "Registration failed" | Service worker file not found | Check `/service-worker.js` exists |
| "VAPID key mismatch" | Server keys changed | Unsubscribe and resubscribe |

### 10. Verify Server-Side

Check the Go backend logs for errors:

```bash
# In terminal
docker logs -f <container-name>
# Look for lines like:
# "Successfully sent push notification"
# "Failed to send notification: <error>"
```

Common server errors:
- VAPID keys not generated
- Database connection issues
- Subscription endpoint unreachable

### 11. Reset Everything

If nothing works, complete reset:

```javascript
// 1. Unregister all service workers
navigator.serviceWorker.getRegistrations().then(regs => {
  regs.forEach(reg => reg.unregister());
});

// 2. Clear all site data
// DevTools → Application → Clear storage → "Clear site data"

// 3. Reload page and try again
location.reload();
```

## Production Checklist

Before deploying to production:

- [ ] HTTPS certificate installed and working
- [ ] Service worker accessible at `/service-worker.js`
- [ ] VAPID keys generated and stored in database
- [ ] Notification permission requested only after user action
- [ ] Error handling for unsupported browsers
- [ ] Fallback UI for denied permissions
- [ ] Test across different browsers
- [ ] Test notification unsubscribe flow
- [ ] Monitor server logs for delivery failures

## Still Not Working?

1. Check this project's GitHub issues
2. Test with a minimal example (see `/api/webpush/README.md`)
3. Verify browser version supports Web Push API
4. Try in a different environment (different machine/network)
5. Check if corporate firewall/antivirus is blocking notifications

## Debugging Resources

- [MDN Web Push API](https://developer.mozilla.org/en-US/docs/Web/API/Push_API)
- [Chrome DevTools Service Worker Debugging](https://developer.chrome.com/docs/devtools/progressive-web-apps/)
- [Firefox Service Worker Debugging](https://firefox-source-docs.mozilla.org/devtools-user/service_worker_debugging/)
- [Web Push Testing Tools](https://web-push-codelab.glitch.me/)
