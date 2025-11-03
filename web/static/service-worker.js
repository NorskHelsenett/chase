self.addEventListener('install', () => {
	self.skipWaiting();
});

self.addEventListener('activate', (event) => {
	event.waitUntil(self.clients.claim());
});

self.addEventListener('push', (event) => {
	if (!event.data) {
		return;
	}

	let payload;
	try {
		payload = event.data.json();
	} catch {
		payload = { title: 'Notification', body: event.data.text() };
	}

	const title = payload.title || 'Notification';

	// Ensure URL is in the data object for the notification click handler
	const notificationData = payload.data || {};
	if (payload.url && !notificationData.url) {
		notificationData.url = payload.url;
	}

	const options = {
		body: payload.body,
		icon: payload.icon || '/images/passkey-hero-aurora.svg',
		badge: payload.badge || '/images/passkey-hero.svg',
		image: payload.image,
		tag: payload.tag,
		data: notificationData
	};

	event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
	event.notification.close();

	// Get the target URL from notification data (will be /notification/[id])
	let targetUrl = event.notification.data?.url || '/dashboard';

	// Handle action-specific behavior
	if (event.action === 'dismiss') {
		// Just close the notification, don't navigate
		return;
	}

	// Ensure the URL is absolute for this origin
	const baseUrl = self.registration.scope;
	if (!targetUrl.startsWith('http')) {
		targetUrl = new URL(targetUrl, baseUrl).href;
	}

	event.waitUntil(
		self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
			// Try to find an existing window and navigate it
			for (const client of clientList) {
				if (client.url.startsWith(baseUrl) && 'focus' in client) {
					return client.focus().then(() => client.navigate(targetUrl));
				}
			}

			// If no existing window, open a new one
			if (self.clients.openWindow) {
				return self.clients.openWindow(targetUrl);
			}
		})
	);
});
