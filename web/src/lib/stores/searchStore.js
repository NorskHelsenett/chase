import { writable } from 'svelte/store';

const STORAGE_KEY = 'search_history';

function createSearchStore() {
	// Load initial state from localStorage
	const storedHistory =
		typeof localStorage !== 'undefined'
			? JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]')
			: [];

	const { subscribe, set, update } = writable(storedHistory);

	return {
		subscribe,
		addSearch: (query, results) => {
			update((history) => {
				const newHistory = [
					{
						query,
						results,
						timestamp: Date.now()
					},
					...history
				].slice(0, 100); // Keep last 100 searches

				// Save to localStorage
				if (typeof localStorage !== 'undefined') {
					localStorage.setItem(STORAGE_KEY, JSON.stringify(newHistory));
				}
				return newHistory;
			});
		},
		clear: () => {
			set([]);
			if (typeof localStorage !== 'undefined') {
				localStorage.removeItem(STORAGE_KEY);
			}
		}
	};
}

export const searchHistory = createSearchStore();
