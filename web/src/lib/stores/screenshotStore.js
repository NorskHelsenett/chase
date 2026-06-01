import { writable } from 'svelte/store';

/** @typedef {'loaded' | 'failed' | 'blank'} ScreenshotState */

// Client-side screenshot load status, keyed by server url.
// 'loaded' = a real thumbnail rendered, 'failed' = image errored,
// 'blank' = image loaded but is all-white (an empty/blank capture).
// Populated lazily by LazyScreenshot as cards enter the viewport.
/** @type {import('svelte/store').Writable<Record<string, ScreenshotState>>} */
export const screenshotStatus = writable({});

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
