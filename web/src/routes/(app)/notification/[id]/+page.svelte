<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	onMount(() => {
		// This is a fallback - the +page.ts should handle the redirect
		// But if somehow we end up here, redirect to dashboard
		setTimeout(() => {
			goto('/dashboard');
		}, 100);
	});
</script>

<div class="flex items-center justify-center min-h-screen">
	<div class="text-center">
		<div
			class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 dark:border-gray-100"
		></div>
		<p class="mt-4 text-gray-600 dark:text-gray-400">Redirecting...</p>
	</div>
</div>
