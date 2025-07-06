/** @type {import('./$types').PageLoad} */
export function load({ params }) {
	return {
		query: params.query?.replace(/^(http|https):\/\//i, '') || ''
	};
}
