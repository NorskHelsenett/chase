import { derived, get, writable } from 'svelte/store';
import {
	getNotificationHistory,
	markNotificationRead as markNotificationReadRequest,
	dismissNotification as dismissNotificationRequest,
	dismissAllNotifications as dismissAllNotificationsRequest
} from './pushClient.js';

export type NotificationMetadata = Record<string, unknown>;

export interface NotificationEvent {
	id: number;
	title: string;
	body: string;
	eventType: string;
	url?: string;
	serverId?: number | null;
	metadata: NotificationMetadata | null;
	serverUrl?: string | null;
	sent: boolean;
	sentAt?: string | null;
	createdAt: string;
	read: boolean;
	readAt?: string | null;
	dismissed: boolean;
	dismissedAt?: string | null;
}

const notifications = writable<NotificationEvent[]>([]);
const isLoading = writable(false);
const loadError = writable<string | null>(null);
const hasLoaded = writable(false);

let loadPromise: Promise<void> | null = null;

function parseMetadata(raw: unknown): NotificationMetadata | null {
	if (typeof raw !== 'string') {
		return raw && typeof raw === 'object' ? (raw as NotificationMetadata) : null;
	}

	if (!raw.trim()) {
		return null;
	}

	try {
		return JSON.parse(raw) as NotificationMetadata;
	} catch (error) {
		console.warn('Failed to parse notification metadata:', error);
		return null;
	}
}

function transformNotification(raw: any): NotificationEvent {
	const parsedMetadata = parseMetadata(raw.metadata);
	const serverUrl =
		typeof parsedMetadata?.serverUrl === 'string' && parsedMetadata.serverUrl.trim()
			? (parsedMetadata.serverUrl as string)
			: null;

	return {
		id: raw.id,
		title: raw.title,
		body: raw.body,
		eventType: raw.event_type,
		url: raw.url || '',
		serverId: raw.server_id ?? null,
		metadata: parsedMetadata,
		serverUrl,
		sent: Boolean(raw.sent),
		sentAt: raw.sent_at ?? null,
		createdAt: raw.created_at,
		read: Boolean(raw.read),
		readAt: raw.read_at ?? null,
		dismissed: Boolean(raw.dismissed),
		dismissedAt: raw.dismissed_at ?? null
	};
}

function cloneNotification(notification: NotificationEvent): NotificationEvent {
	return {
		...notification,
		metadata: notification.metadata ? { ...notification.metadata } : null,
		serverUrl: notification.serverUrl ?? null
	};
}

export async function loadNotifications(force = false): Promise<void> {
	if (loadPromise && !force) {
		return loadPromise;
	}

	isLoading.set(true);
	loadError.set(null);

	loadPromise = (async () => {
		try {
			const history = await getNotificationHistory(100);
			const items = history.map(transformNotification);
			notifications.set(items);
		} catch (error) {
			const message =
				error instanceof Error ? error.message : 'Failed to load notifications';
			loadError.set(message);
			throw error;
		} finally {
			isLoading.set(false);
			hasLoaded.set(true);
			loadPromise = null;
		}
	})();

	return loadPromise;
}

function updateNotificationEntry(
	id: number,
	updater: (current: NotificationEvent) => NotificationEvent
): { previous: NotificationEvent | null; didUpdate: boolean } {
	let previous: NotificationEvent | null = null;
	let didUpdate = false;

	notifications.update((items) =>
		items.map((item) => {
			if (item.id !== id) {
				return item;
			}

			previous = cloneNotification(item);
			const next = updater(item);
			didUpdate = true;
			return next;
		})
	);

	return { previous, didUpdate };
}

export async function markNotificationAsRead(
	id: number,
	options: { skipRequest?: boolean } = {}
): Promise<void> {
	const { previous, didUpdate } = updateNotificationEntry(id, (item) => {
		if (item.read) {
			return item;
		}

		const timestamp = new Date().toISOString();
		return {
			...item,
			read: true,
			readAt: item.readAt ?? timestamp
		};
	});

	if (!didUpdate || options.skipRequest) {
		return;
	}

	try {
		const response = await markNotificationReadRequest(id);
		const updated = response?.notification ? transformNotification(response.notification) : null;
		if (updated) {
			notifications.update((items) =>
				items.map((item) => (item.id === id ? updated : item))
			);
		}
	} catch (error) {
		if (previous) {
			notifications.update((items) =>
				items.map((item) => (item.id === id ? previous : item))
			);
		}
		throw error;
	}
}

export async function dismissNotification(id: number): Promise<void> {
	let removed: NotificationEvent | null = null;
	let index = -1;

	notifications.update((items) => {
		index = items.findIndex((item) => item.id === id);
		if (index === -1) {
			return items;
		}

		removed = cloneNotification(items[index]);
		return [...items.slice(0, index), ...items.slice(index + 1)];
	});

	if (index === -1) {
		return;
	}

	try {
		await dismissNotificationRequest(id);
	} catch (error) {
		if (removed) {
			notifications.update((items) => {
				const next = [...items];
				next.splice(index, 0, removed as NotificationEvent);
				return next;
			});
		}
		throw error;
	}
}

export async function dismissAllNotifications(): Promise<void> {
	const previous = get(notifications);
	if (previous.length === 0) {
		return;
	}

	notifications.set([]);

	try {
		await dismissAllNotificationsRequest();
	} catch (error) {
		notifications.set(previous);
		throw error;
	}
}

export function resetNotifications(): void {
	notifications.set([]);
	hasLoaded.set(false);
	loadError.set(null);
	loadPromise = null;
}

export const unreadCount = derived(notifications, ($notifications) =>
	$notifications.reduce((count, notification) => (notification.read ? count : count + 1), 0)
);

export { notifications, isLoading, loadError, hasLoaded };
