import { error } from '@sveltejs/kit';

const API_URL = process.env.API_URL || 'http://localhost:8080';

/** @type {import('./$types').RequestHandler} */
export async function GET({ params, url, fetch }) {
	const domain = params.path;
	
	if (!domain) {
		throw error(400, 'Domain parameter is required');
	}

	// Build query parameters with fullSize=true as default
	const queryParams = new URLSearchParams(url.searchParams);
	
	// Set fullSize=true by default if not explicitly set
	if (!queryParams.has('fullSize') && !queryParams.has('fullsize')) {
		queryParams.set('fullSize', 'true');
	}
	
	const apiUrl = `${API_URL}/api/screenshot/${domain}${queryParams.toString() ? '?' + queryParams.toString() : ''}`;

	try {
		const response = await fetch(apiUrl);
		
		// Forward the response from the API
		const contentType = response.headers.get('content-type');
		const cacheControl = response.headers.get('cache-control');
		const screenshotCached = response.headers.get('x-screenshot-cached');
		
		if (!response.ok) {
			// If API returns error, forward it
			const errorData = await response.json();
			throw error(response.status, errorData.error || 'Screenshot service error');
		}

		// Get the image data
		const imageBuffer = await response.arrayBuffer();
		
		// Return the image with proper headers
		return new Response(imageBuffer, {
			status: 200,
			headers: {
				'Content-Type': contentType || 'image/png',
				'Cache-Control': cacheControl || 'public, max-age=3600',
				...(screenshotCached && { 'X-Screenshot-Cached': screenshotCached })
			}
		});
	} catch (err) {
		console.error(`Screenshot proxy error for ${domain}:`, err);
		
		// @ts-ignore - error type handling
		if (err.status) {
			throw err;
		}
		
		throw error(503, 'Screenshot service temporarily unavailable');
	}
}
