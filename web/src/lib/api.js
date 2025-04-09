// src/lib/api.js
const BASE_URL = '/api';

export async function login() {
	const response = await fetch(`${BASE_URL}/login`);
	if (response.ok) {
		const data = await response.json();
		// Redirect to the OAuth provider URL
		window.location.href = data.url;
	} else {
		throw new Error('Failed to initiate login');
	}
}

export async function logout() {
	const response = await fetch(`${BASE_URL}/logout`, {
		credentials: 'include'
	});
	if (response.ok) {
		return true;
	} else {
		throw new Error('Failed to logout');
	}
}

export async function getProtectedData() {
	const response = await fetch(`${BASE_URL}/profile`, {
		credentials: 'include'
	});
	if (response.ok) {
		return response.json();
	} else {
		throw new Error('Failed to get protected data');
	}
}

export async function updateVisitedServers(visitedServers) {
	const response = await fetch(`${BASE_URL}/profile`, {
		method: 'PATCH',
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ visited_servers: visitedServers })
	});
	
	if (response.ok) {
		return response.json();
	} else {
		throw new Error('Failed to update visited servers');
	}
}
