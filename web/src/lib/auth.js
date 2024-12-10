import { writable, derived } from 'svelte/store';
import { getProtectedData, logout as apiLogout } from './api';
import md5 from 'md5';

export const user = writable(null);

// Create a derived store for login status
export const isLoggedIn = derived(user, $user => !!$user);

export async function initializeAuth() {
    try {
        const userData = await getProtectedData();
        user.set(userData);
        return true;
    } catch (error) {
        console.error('Failed to initialize auth:', error);
        return false;
    }
}

export async function getUserInfo() {
    return new Promise((resolve) => {
        user.subscribe((value) => {
            resolve(value);
        })();
    });
}

export async function logout() {
    try {
        await apiLogout();
        user.set(null);
    } catch (error) {
        console.error('Logout failed:', error);
        throw error;
    }
}

export function getEmailHash(email) {
    return md5(email.trim().toLowerCase());
}