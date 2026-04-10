import { writable } from 'svelte/store';
import { browser } from '$app/environment';

/**
 * Per-server ping data from SSE.
 * Map<serverID, { latest, days, expectedStatus }>
 *   latest: { server_id, status_code, response_time_ms, error?, timestamp }
 *   days: [{ date, total, successful, uptime }]  (last 14 days, oldest first)
 *   expectedStatus: number
 */
export const pingData = writable(new Map());

let eventSource = null;

export function connectPingSSE() {
	if (!browser || eventSource) return;

	eventSource = new EventSource('/api/servers/pings/stream');

	eventSource.addEventListener('init', (e) => {
		const data = JSON.parse(e.data);
		pingData.update((map) => {
			map.set(data.server_id, {
				latest: data.latest,
				days: data.days || [],
				expectedStatus: data.expected_status
			});
			return new Map(map);
		});
	});

	eventSource.addEventListener('ping', (e) => {
		const ping = JSON.parse(e.data);
		pingData.update((map) => {
			const existing = map.get(ping.server_id) || { latest: null, days: [], expectedStatus: ping.expected_status };

			// Update expected status if provided
			if (ping.expected_status) {
				existing.expectedStatus = ping.expected_status;
			}

			// Update latest ping
			existing.latest = ping;

			// Update today's summary
			const today = new Date(ping.timestamp).toISOString().split('T')[0];
			const todayIdx = existing.days.findIndex((d) => d.date === today);
			const isSuccess =
				ping.status_code > 0 && ping.status_code === existing.expectedStatus && !ping.error;

			if (todayIdx >= 0) {
				existing.days[todayIdx].total += 1;
				if (isSuccess) existing.days[todayIdx].successful += 1;
				existing.days[todayIdx].uptime =
					(existing.days[todayIdx].successful / existing.days[todayIdx].total) * 100;
			} else {
				existing.days.push({
					date: today,
					total: 1,
					successful: isSuccess ? 1 : 0,
					uptime: isSuccess ? 100 : 0
				});
				// Keep only last 14 days
				if (existing.days.length > 14) {
					existing.days = existing.days.slice(-14);
				}
			}

			map.set(ping.server_id, { ...existing });
			return new Map(map);
		});
	});

	eventSource.onerror = () => {
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
