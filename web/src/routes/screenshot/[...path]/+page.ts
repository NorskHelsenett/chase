/** @type {import('./$types').PageLoad} */
export function load({ params, url }) {
	const domain = params.path;
	const searchParams = url.searchParams;
	
	// Default to fullSize=true if not specified
	const fullSize = searchParams.get('fullSize') || searchParams.get('fullsize') || 'true';
	const cached = searchParams.get('cached') || 'false';
	const waitTime = searchParams.get('waitTime') || searchParams.get('waittime') || '3';
	
	// Build query string
	const queryParams = new URLSearchParams();
	queryParams.set('fullSize', fullSize);
	if (cached === 'true') {
		queryParams.set('cached', 'true');
	}
	queryParams.set('waitTime', waitTime);
	
	const imageSrc = `/api/screenshot/${domain}?${queryParams.toString()}`;
	
	return {
		domain,
		imageSrc
	};
}
