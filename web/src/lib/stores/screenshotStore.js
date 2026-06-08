import { writable } from 'svelte/store';
import { browser } from '$app/environment';

/** @typedef {'loaded' | 'failed' | 'blank'} ScreenshotState */

const STORAGE_KEY = 'chase-screenshot-status';

/** @returns {Record<string, ScreenshotState>} */
function loadInitial() {
	if (!browser) return {};
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		return raw ? JSON.parse(raw) : {};
	} catch {
		return {};
	}
}

// Client-side screenshot load status, keyed by server url.
// 'loaded' = a real thumbnail rendered, 'failed' = image errored,
// 'blank' = image loaded but is all-white (an empty/blank capture).
// Populated lazily by LazyScreenshot as cards enter the viewport, and
// persisted to localStorage so the Online filter can apply known statuses
// on the next page load (the grid only re-evaluates on (re)load, never
// while scrolling).
/** @type {import('svelte/store').Writable<Record<string, ScreenshotState>>} */
export const screenshotStatus = writable(loadInitial());

if (browser) {
	screenshotStatus.subscribe((map) => {
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
		} catch {
			// Ignore quota / serialization errors — persistence is best-effort.
		}
	});
}

/**
 * @param {string} url
 * @param {ScreenshotState} status
 */
export function reportScreenshotStatus(url, status) {
	screenshotStatus.update((map) => {
		if (map[url] === status) return map;
		return { ...map, [url]: status };
	});
}
