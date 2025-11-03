import { error, redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, fetch }) => {
    const notificationId = params.id;

    try {
        const response = await fetch(`/api/notification/${notificationId}`);

        if (!response.ok) {
            if (response.status === 404) {
                throw error(404, 'Notification not found');
            }
            throw error(response.status, 'Failed to load notification');
        }

        const data = await response.json();

        // Redirect to the target URL
        if (data.redirect) {
            throw redirect(302, data.redirect);
        }

        // Fallback to dashboard if no redirect URL
        throw redirect(302, '/dashboard');
    } catch (err) {
        // If it's already a redirect or error, re-throw it
        if (err && typeof err === 'object' && ('status' in err || 'location' in err)) {
            throw err;
        }

        console.error('Failed to load notification:', err);
        // Fallback to dashboard on error
        throw redirect(302, '/dashboard');
    }
};
