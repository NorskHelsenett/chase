import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

// Store: Map<serverID, PingEvent[]>
export const pingData = writable(new Map());

let eventSource = null;

export function connectPingSSE() {
	if (!browser || eventSource) return;

	eventSource = new EventSource('/api/servers/pings/stream');

	eventSource.addEventListener('init', (e) => {
		const { server_id, pings } = JSON.parse(e.data);
		pingData.update((map) => {
			map.set(server_id, pings);
			return new Map(map);
		});
	});

	eventSource.addEventListener('ping', (e) => {
		const ping = JSON.parse(e.data);
		pingData.update((map) => {
			const existing = map.get(ping.server_id) || [];
			// Prepend new ping, keep max 10
			const updated = [ping, ...existing].slice(0, 10);
			map.set(ping.server_id, updated);
			return new Map(map);
		});
	});

	eventSource.onerror = () => {
		// Reconnect after 5 seconds on error
		disconnectPingSSE();
		setTimeout(connectPingSSE, 5000);
	};
}

export function disconnectPingSSE() {
	if (eventSource) {
		eventSource.close();
		eventSource = null;
	}
}

/**
 * Get ping results for a specific server from the store.
 * Returns the array or empty array.
 */
export function getPingsForServer(serverID) {
	const map = get(pingData);
	return map.get(serverID) || [];
}
