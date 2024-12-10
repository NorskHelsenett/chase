import { json } from '@sveltejs/kit';

/** @type {import('./$types').RequestHandler} */
export async function GET({ url }) {
    const query = url.searchParams.get('q');
    
    // Replace this with your actual search logic
    const results = [
        { title: `Result 1 for ${query}` },
        { title: `Result 2 for ${query}` }
    ];
    
    return json(results);
}