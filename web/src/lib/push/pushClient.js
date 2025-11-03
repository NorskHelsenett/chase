// Web Push Notification Client
// Handles service worker registration and push subscription management

/**
 * Initialize push notifications
 * Call this when your app loads
 */
export async function initPushNotifications() {
	if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
		console.warn('Push notifications are not supported in this browser');
		return null;
	}

	try {
		const registration = await navigator.serviceWorker.register('/service-worker.js');
		return registration;
	} catch (error) {
		console.error('Service worker registration failed:', error);
		return null;
	}
}

/**
 * Request notification permission from the user
 */
export async function requestNotificationPermission() {
	if (!('Notification' in window)) {
		return 'unsupported';
	}

	if (Notification.permission === 'granted') {
		return 'granted';
	}

	if (Notification.permission === 'denied') {
		return 'denied';
	}

	const permission = await Notification.requestPermission();
	return permission;
}

/**
 * Get the VAPID public key from the server
 */
export async function getVAPIDPublicKey() {
	try {
		const response = await fetch('/api/push/vapid-public-key');
		if (!response.ok) {
			throw new Error('Failed to get VAPID public key');
		}
		const data = await response.json();
		return data.publicKey;
	} catch (error) {
		console.error('Error getting VAPID public key:', error);
		throw error;
	}
}

/**
 * Subscribe to push notifications
 * Returns the subscription object if successful
 */
export async function subscribeToPush() {
	try {
		// Check permission
		const permission = await requestNotificationPermission();
		if (permission !== 'granted') {
			throw new Error('Notification permission not granted');
		}

		// Get service worker registration
		const registration = await navigator.serviceWorker.ready;

		// Get VAPID public key
		const vapidPublicKey = await getVAPIDPublicKey();

		// Convert base64 to Uint8Array
		const applicationServerKey = urlBase64ToUint8Array(vapidPublicKey);

		// Subscribe to push
		const subscription = await registration.pushManager.subscribe({
			userVisibleOnly: true,
			applicationServerKey: applicationServerKey
		});

		// Send subscription to server
		await sendSubscriptionToServer(subscription);

		console.log('Push subscription successful:', subscription);
		return subscription;
	} catch (error) {
		console.error('Error subscribing to push:', error);
		throw error;
	}
}

/**
 * Unsubscribe from push notifications
 */
export async function unsubscribeFromPush() {
	try {
		const registration = await navigator.serviceWorker.ready;
		const subscription = await registration.pushManager.getSubscription();

		if (subscription) {
			// Unsubscribe from browser
			await subscription.unsubscribe();

			// Remove from server
			await removeSubscriptionFromServer(subscription);

			console.log('Successfully unsubscribed from push');
			return true;
		}
		return false;
	} catch (error) {
		console.error('Error unsubscribing from push:', error);
		throw error;
	}
}

/**
 * Get current push subscription status
 */
export async function getPushSubscription() {
	try {
		const registration = await navigator.serviceWorker.ready;
		const subscription = await registration.pushManager.getSubscription();
		return subscription;
	} catch (error) {
		console.error('Error getting push subscription:', error);
		return null;
	}
}

/**
 * Check if push is currently subscribed
 */
export async function isPushSubscribed() {
	const subscription = await getPushSubscription();
	return subscription !== null;
}

/**
 * Send subscription to server
 */
async function sendSubscriptionToServer(subscription) {
	const subscriptionJSON = subscription.toJSON();

	const response = await fetch('/api/push/subscribe', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({
			endpoint: subscriptionJSON.endpoint,
			keys: {
				auth: subscriptionJSON.keys.auth,
				p256dh: subscriptionJSON.keys.p256dh
			}
		})
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.error || 'Failed to save subscription');
	}

	return await response.json();
}

/**
 * Remove subscription from server
 */
async function removeSubscriptionFromServer(subscription) {
	const subscriptionJSON = subscription.toJSON();

	const response = await fetch('/api/push/unsubscribe', {
		method: 'DELETE',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({
			endpoint: subscriptionJSON.endpoint
		})
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.error || 'Failed to remove subscription');
	}

	return await response.json();
}

/**
 * Get notification preferences from server
 */
export async function getNotificationPreferences() {
	try {
		const response = await fetch('/api/push/preferences');
		if (!response.ok) {
			throw new Error('Failed to get preferences');
		}
		const data = await response.json();
		return data.preferences;
	} catch (error) {
		console.error('Error getting notification preferences:', error);
		throw error;
	}
}

/**
 * Update notification preferences on server
 */
export async function updateNotificationPreferences(preferences) {
	try {
		const response = await fetch('/api/push/preferences', {
			method: 'PUT',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify(preferences)
		});

		if (!response.ok) {
			throw new Error('Failed to update preferences');
		}

		return await response.json();
	} catch (error) {
		console.error('Error updating notification preferences:', error);
		throw error;
	}
}

/**
 * Get available notification event types
 */
export async function getEventTypes() {
	try {
		const response = await fetch('/api/push/event-types');
		if (!response.ok) {
			throw new Error('Failed to get event types');
		}
		const data = await response.json();
		return data.eventTypes;
	} catch (error) {
		console.error('Error getting event types:', error);
		throw error;
	}
}

/**
 * Get notification history
 */
export async function getNotificationHistory(limit = 50) {
	try {
		const response = await fetch(`/api/push/history?limit=${limit}`);
		if (!response.ok) {
			throw new Error('Failed to get notification history');
		}
		const data = await response.json();
		return data.history;
	} catch (error) {
		console.error('Error getting notification history:', error);
		throw error;
	}
}

/**
 * Send a test notification
 */
export async function sendTestNotification() {
	try {
		const response = await fetch('/api/push/test', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			}
		});

		if (!response.ok) {
			const error = await response.json();
			throw new Error(error.error || 'Failed to send test notification');
		}

		return await response.json();
	} catch (error) {
		console.error('Error sending test notification:', error);
		throw error;
	}
}

/**
 * Utility: Convert base64 VAPID key to Uint8Array
 */
function urlBase64ToUint8Array(base64String) {
	const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
	const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');

	const rawData = window.atob(base64);
	const outputArray = new Uint8Array(rawData.length);

	for (let i = 0; i < rawData.length; ++i) {
		outputArray[i] = rawData.charCodeAt(i);
	}
	return outputArray;
}
