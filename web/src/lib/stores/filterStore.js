import { writable } from 'svelte/store';
import { browser } from '$app/environment';

const stored = browser && localStorage.getItem('chase-filter-status');
export const statusFilter = writable(stored || 'all');

statusFilter.subscribe((value) => {
	if (browser) {
		localStorage.setItem('chase-filter-status', value);
	}
});
