// Service Worker for Push Notifications

// Install event
self.addEventListener('install', (event) => {
	self.skipWaiting();
});

// Activate event
self.addEventListener('activate', (event) => {
	event.waitUntil(clients.claim());
});

// Push event - handle incoming push notifications
self.addEventListener('push', (event) => {
	let notification = {
		title: 'New Notification',
		body: 'You have a new notification',
		icon: '/icon-192.png',
		badge: '/badge.png',
		vibrate: [200, 100, 200],
		requireInteraction: false,
		data: {}
	};

	try {
		if (event.data) {
			const data = event.data.json();
			notification = {
				title: data.title || notification.title,
				body: data.body || notification.body,
				icon: data.icon || notification.icon,
				badge: data.badge || notification.badge,
				image: data.image,
				tag: data.tag || 'notification',
				requireInteraction: data.requireInteraction || false,
				vibrate: data.vibrate || notification.vibrate,
				data: data.data || {},
				actions: data.actions || []
			};
		}
	} catch (error) {
		console.error('Error parsing push data:', error);
	}

	event.waitUntil(
		self.registration.showNotification(notification.title, {
			body: notification.body,
			icon: notification.icon,
			badge: notification.badge,
			image: notification.image,
			tag: notification.tag,
			requireInteraction: notification.requireInteraction,
			vibrate: notification.vibrate,
			data: notification.data,
			actions: notification.actions
		})
	);
});

// Notification click event
self.addEventListener('notificationclick', (event) => {
	console.log('Notification clicked:', event);

	event.notification.close();

	// Handle action buttons
	if (event.action) {
		console.log('Action clicked:', event.action);
		
		if (event.action === 'dismiss') {
			return;
		}
	}

	// Get the URL to open
	const urlToOpen = event.notification.data?.url || '/';

	// Open or focus the app
	const promiseChain = clients.matchAll({
		type: 'window',
		includeUncontrolled: true
	}).then((windowClients) => {
		// Check if there's already a window open
		for (let i = 0; i < windowClients.length; i++) {
			const client = windowClients[i];
			if (client.url === urlToOpen && 'focus' in client) {
				return client.focus();
			}
		}

		// No window open, open a new one
		if (clients.openWindow) {
			return clients.openWindow(urlToOpen);
		}
	});

	event.waitUntil(promiseChain);
});

// Notification close event
self.addEventListener('notificationclose', (event) => {
	console.log('Notification closed:', event);
	
	// You can track notification dismissals here if needed
	// For example, send analytics to your server
});

// Background sync for failed notifications (optional)
self.addEventListener('sync', (event) => {
	console.log('Background sync:', event);
	
	if (event.tag === 'sync-notifications') {
		event.waitUntil(syncNotifications());
	}
});

async function syncNotifications() {
	// Implement sync logic if needed
	console.log('Syncing notifications...');
}

// Message event - for communication with the app
self.addEventListener('message', (event) => {
	console.log('Service Worker received message:', event.data);

	if (event.data && event.data.type === 'SKIP_WAITING') {
		self.skipWaiting();
	}
});
