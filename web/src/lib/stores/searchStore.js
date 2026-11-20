import { writable, get } from 'svelte/store';

const STORAGE_KEY = 'search_history';
const CACHE_DURATION_MS = 5 * 60 * 1000; // 5 minutes in milliseconds

function createSearchStore() {
	// Load initial state from localStorage
	let storedHistory = [];
	try {
		if (typeof localStorage !== 'undefined') {
			const parsed = JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]');
			if (Array.isArray(parsed)) storedHistory = parsed;
		}
	} catch {
		// Corrupt or unreadable value: start with an empty history.
	}

	const { subscribe, set, update } = writable(storedHistory);

	return {
		subscribe,
		addSearch: (query, results) => {
			update((history) => {
				// Normalize query for comparison
				const normalizedQuery = query.toLowerCase().trim();

				// Remove existing entry for this query if it exists
				const filteredHistory = history.filter(
					(item) => item.query.toLowerCase().trim() !== normalizedQuery
				);

				const newHistory = [
					{
						query,
						results,
						timestamp: Date.now()
					},
					...filteredHistory
				].slice(0, 100); // Keep last 100 searches

				// Save to localStorage
				if (typeof localStorage !== 'undefined') {
					try {
						localStorage.setItem(STORAGE_KEY, JSON.stringify(newHistory));
					} catch {
						// Storage unavailable: history stays in memory only.
					}
				}
				return newHistory;
			});
		},
		/**
		 * Get a recent search result if it exists and is fresh (within 5 minutes)
		 * @param {string} query - The search query to look up
		 * @returns {{ results: object, timestamp: number } | null} - The cached result or null
		 */
		getRecentSearch: (query) => {
			const history = get({ subscribe });
			const normalizedQuery = query.toLowerCase().trim();
			const now = Date.now();

			const cached = history.find((item) => item.query.toLowerCase().trim() === normalizedQuery);

			if (cached && now - cached.timestamp < CACHE_DURATION_MS) {
				return {
					results: cached.results,
					timestamp: cached.timestamp
				};
			}

			return null;
		},
		clear: () => {
			set([]);
			if (typeof localStorage !== 'undefined') {
				try {
					localStorage.removeItem(STORAGE_KEY);
				} catch {
					// Storage unavailable: history stays in memory only.
				}
			}
		}
	};
}

export const searchHistory = createSearchStore();
