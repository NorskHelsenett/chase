import type { Server } from '$lib/models';
import { get } from 'svelte/store';
import { pingData } from '$lib/stores/pingStore';

/**
 * Resolve effective status using SSE ping data when available,
 * falling back to the API-provided status.
 *
 * This ensures counters, filters, and visuals all agree on whether
 * a server is "up" or "down", avoiding the mismatch where the API
 * marks a server as "stale" but real-time SSE shows it responding.
 */
export function getEffectiveStatus(server: Server, pingMap?: Map<number, any>): 'up' | 'down' {
	const map = pingMap ?? get(pingData);
	const info = map.get(server.ID);
	if (info?.latest) {
		const s = info.latest.status_code;
		return s > 0 && s === server.expected_status && !info.latest.error ? 'up' : 'down';
	}
	// No live SSE data yet — fall back to what the list endpoint already reported.
	// The last observed check is the source of truth: a host whose most recent probe
	// matched the expected status is up, even if that probe is "stale". With many
	// servers on a fixed cadence most are stale at any instant, so gating purely on
	// status === 'up' wrongly hides the bulk of healthy hosts.
	if (server.last_status_code != null && server.last_status_code !== 0) {
		return server.last_status_code === server.expected_status ? 'up' : 'down';
	}
	return server.status === 'up' ? 'up' : 'down';
}
