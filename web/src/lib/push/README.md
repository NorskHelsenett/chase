# Web Push Notifications - Frontend

SvelteKit components and utilities for Web Push notifications.

## Files in this Package

- `pushClient.js` - Core JavaScript utilities for push subscription management
- `PushNotificationSettings.svelte` - Ready-to-use settings UI component
- `/static/service-worker.js` - Service worker for handling push events (copy to your `/static` folder)

## Quick Setup

### 1. Copy Files

```bash
# Copy these files to your SvelteKit project:
cp pushClient.js /your-project/src/lib/push/
cp PushNotificationSettings.svelte /your-project/src/lib/push/
cp ../static/service-worker.js /your-project/static/
```

### 2. Use the Component

In any Svelte page (e.g., settings page):

```svelte
<script>
	import PushNotificationSettings from '$lib/push/PushNotificationSettings.svelte';
</script>

<PushNotificationSettings />
```

That's it! The component handles everything:
- Requesting notification permissions
- Subscribing/unsubscribing
- Managing preferences
- Sending test notifications

### 3. Initialize on App Load (Optional)

To initialize the service worker when your app loads, add this to your root `+layout.svelte`:

```svelte
<script>
	import { onMount } from 'svelte';
	import { initPushNotifications } from '$lib/push/pushClient.js';

	onMount(async () => {
		await initPushNotifications();
	});
</script>
```

## Advanced Usage

### Custom Notification UI

If you want to build your own UI instead of using the component:

```svelte
<script>
	import {
		subscribeToPush,
		unsubscribeFromPush,
		isPushSubscribed,
		getNotificationPreferences,
		updateNotificationPreferences
	} from '$lib/push/pushClient.js';

	let isSubscribed = false;

	async function checkStatus() {
		isSubscribed = await isPushSubscribed();
	}

	async function subscribe() {
		try {
			await subscribeToPush();
			isSubscribed = true;
		} catch (error) {
			console.error('Subscribe failed:', error);
		}
	}

	async function unsubscribe() {
		try {
			await unsubscribeFromPush();
			isSubscribed = false;
		} catch (error) {
			console.error('Unsubscribe failed:', error);
		}
	}
</script>

<button on:click={isSubscribed ? unsubscribe : subscribe}>
	{isSubscribed ? 'Disable' : 'Enable'} Notifications
</button>
```

### Available Functions

#### `initPushNotifications()`
Registers the service worker. Returns the registration object.

#### `requestNotificationPermission()`
Requests permission from the user. Returns: `'granted'`, `'denied'`, or `'default'`.

#### `subscribeToPush()`
Subscribes the user to push notifications. Automatically requests permission if needed.

#### `unsubscribeFromPush()`
Unsubscribes the user from push notifications.

#### `isPushSubscribed()`
Returns `true` if currently subscribed, `false` otherwise.

#### `getNotificationPreferences()`
Gets the user's notification preferences from the server.
Returns an object like: `{ "server_offline": true, "server_online": false, ... }`

#### `updateNotificationPreferences(preferences)`
Updates notification preferences on the server.

```js
await updateNotificationPreferences({
	"server_offline": true,
	"server_online": true,
	"server_added": false
});
```

#### `getEventTypes()`
Gets available notification event types with metadata.

#### `sendTestNotification()`
Sends a test notification to verify everything is working.

## Customizing the Service Worker

The service worker in `/static/service-worker.js` can be customized:

### Change Notification Appearance

```js
// In service-worker.js, modify the defaults:
let notification = {
	title: 'Your App Name',
	icon: '/your-icon.png',
	badge: '/your-badge.png',
	// ...
};
```

### Add Custom Click Behavior

```js
// In the 'notificationclick' handler:
self.addEventListener('notificationclick', (event) => {
	// Your custom logic
	const data = event.notification.data;
	
	if (data.type === 'server_offline') {
		// Navigate to specific page
		clients.openWindow('/servers/' + data.serverId);
	}
});
```

## Icons and Images

Create these icons in your `/static` folder:

- `/static/icon-192.png` - Main notification icon (192x192)
- `/static/icon-512.png` - Large icon (512x512)
- `/static/badge.png` - Badge icon (96x96, monochrome)
- `/static/badge-error.png` - Error badge (96x96)
- `/static/badge-success.png` - Success badge (96x96)
- `/static/badge-warning.png` - Warning badge (96x96)

## Browser Support

This implementation works in:
- Chrome/Edge 42+
- Firefox 44+
- Safari 16+ (macOS 13+, iOS 16.4+)
- Opera 29+

## Debugging

### Check Service Worker Status

1. Open DevTools
2. Go to Application > Service Workers
3. Verify your service worker is registered

### Test Notifications

Use the browser console:

```js
// Check if service worker is registered
navigator.serviceWorker.getRegistration().then(reg => console.log(reg));

// Check current subscription
navigator.serviceWorker.ready.then(reg => 
	reg.pushManager.getSubscription().then(sub => console.log(sub))
);

// Check permission status
console.log(Notification.permission);
```

### Common Issues

**"Service worker not found"**
- Ensure `service-worker.js` is in your `/static` folder
- The service worker must be served from the same origin

**"Push not supported"**
- Check if you're using HTTPS (required for push notifications)
- Verify browser compatibility

**"Permission denied"**
- User clicked "Block" - they need to reset permissions in browser settings
- On Safari, check Settings > Notifications

**Notifications not appearing**
- Check Do Not Disturb mode on the device
- Verify the service worker's push event handler is working
- Check browser notification settings

## Security Notes

- Always use HTTPS in production (required for service workers)
- The VAPID public key is public and safe to expose
- Never expose the VAPID private key to the frontend
- Subscriptions include sensitive endpoints - handle with care

## Testing Locally

For local development without HTTPS:

1. Use `localhost` (browsers allow service workers on localhost)
2. Or use a tool like ngrok to get HTTPS locally

```bash
npx serve static -p 5000
# Or with vite dev server
npm run dev
```

## Copying to Other Projects

To use this in another project:

1. Copy the `push/` directory to your project's `src/lib/`
2. Copy `service-worker.js` to your `static/` folder
3. Update API endpoints if your backend is on a different path
4. Customize the UI styles to match your design system

That's it! The code is self-contained and portable.
