<script>
	import { initializeAuth } from '$lib/auth';
	import Navbar from '$lib/components/Navbar.svelte';
	import { connectPingSSE, disconnectPingSSE } from '$lib/stores/pingStore';
	import { onMount, onDestroy } from 'svelte';

	onMount(async () => {
		await initializeAuth();
		connectPingSSE();
	});

	onDestroy(() => {
		disconnectPingSSE();
	});
</script>

<!-- Mobile view -->
<div class="min-h-screen flex flex-col sm:hidden bg-background text-foreground overflow-auto">
	<div class="flex-grow flex items-center justify-center p-4"></div>
</div>

<!-- Desktop view -->
<div class="hidden sm:flex min-h-screen w-full bg-background text-foreground">
	<!-- Navbar -->
	<nav
		class="fixed left-0 top-0 bottom-0 flex-shrink-0 flex flex-col justify-between py-4 bg-background overflow-y-auto z-10"
		style="width: 200px;"
	>
		<Navbar />
	</nav>

	<!-- Main content area -->
	<main class="flex-grow overflow-auto p-4" style="margin-left: 200px;">
		<div class="w-full flex items-center justify-center">
			<slot />
		</div>
	</main>
</div>
