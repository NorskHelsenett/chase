import { writable } from 'svelte/store';
import { browser } from '$app/environment';

let stored = null;
if (browser) {
	try {
		stored = localStorage.getItem('chase-filter-status');
	} catch {
		stored = null;
	}
}
export const statusFilter = writable(stored || 'all');

statusFilter.subscribe((value) => {
	if (browser) {
		try {
			localStorage.setItem('chase-filter-status', value);
		} catch {
			// Storage unavailable (private mode, quota): keep the in-memory value.
		}
	}
});
