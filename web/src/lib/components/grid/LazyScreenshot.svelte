<script>
	import { run } from 'svelte/legacy';

	import { onMount } from 'svelte';

	let imageErrors = $state({});
	let imageLoaded = $state({});

	function checkImage(url) {
		const img = new Image();
		img.onload = () => (imageLoaded[url] = true);
		img.onerror = () => (imageErrors[url] = true);
		img.src = url;
	}

	function lazyCheck(node, url) {
		let currentUrl = url;
		let observer;

		function setupObserver() {
			// Clear previous observer if it exists
			if (observer) {
				observer.disconnect();
			}

			observer = new IntersectionObserver(
				([entry]) => {
					if (entry.isIntersecting) {
						checkImage(currentUrl);
					}
				},
				{
					rootMargin: '100px' // preload a bit earlier
				}
			);

			observer.observe(node);
		}

		// Initial setup
		setupObserver();

		return {
			update(newUrl) {
				if (newUrl !== currentUrl) {
					currentUrl = newUrl;
					setupObserver();
				}
			},
			destroy() {
				if (observer) {
					observer.disconnect();
				}
			}
		};
	}

	// Track if component is visible
	let isVisible = false;

	// Function to check if element is in viewport
	function checkVisibility(node) {
		const observer = new IntersectionObserver(([entry]) => {
			isVisible = entry.isIntersecting;
			if (isVisible && imageUrl) {
				checkImage(imageUrl);
			}
		});

		observer.observe(node);

		return {
			destroy() {
				observer.disconnect();
			}
		};
	}

	let { site, getScreenshotUrl } = $props();

	let prevImageUrl = $state('');
	let imageUrl = $state();

	run(() => {
		imageUrl = getScreenshotUrl(site.url);
		// Reset states when URL changes
		if (imageUrl !== prevImageUrl) {
			delete imageLoaded[prevImageUrl];
			delete imageErrors[prevImageUrl];
			prevImageUrl = imageUrl;
		}
	});
</script>

<!-- Container that triggers lazy load -->
<div use:lazyCheck={imageUrl} class="w-full h-full">
	{#if imageLoaded[imageUrl]}
		<img
			src={imageUrl}
			alt={`Screenshot of ${site.url}`}
			class="absolute inset-0 w-full h-full object-cover transition-all duration-300 group-hover:scale-105 group-hover:brightness-105 rounded-t-xl"
		/>
	{:else if imageErrors[imageUrl]}
		<div
			class="absolute inset-0 flex flex-col items-center justify-center bg-black/70 text-gray-300 rounded-xl p-4 backdrop-blur-sm"
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="32"
				height="32"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
				class="mb-3 text-gray-400 opacity-60"
			>
				<rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect>
				<circle cx="9" cy="9" r="2"></circle>
				<path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
			</svg>
			<span class="text-center font-medium">Failed to load screenshot</span>
			<span class="text-xs text-gray-400 mt-1"
				>{new URL(site.url.startsWith('http') ? site.url : `https://${site.url}`).hostname}</span
			>
		</div>
	{:else}
		<div
			class="absolute inset-0 bg-gradient-to-br from-gray-900/80 to-black/90 rounded-lg flex flex-col items-center justify-center"
		>
			<div class="relative">
				<div
					class="w-12 h-12 border-4 border-t-green-500 border-r-green-400/40 border-b-green-400/20 border-l-green-400/60 rounded-full animate-spin"
				></div>
				<div
					class="absolute inset-0 w-12 h-12 border-4 border-green-500/10 rounded-full animate-pulse"
				></div>
			</div>
			<p class="text-green-400 mt-3 text-sm font-medium">Loading screenshot...</p>
		</div>
	{/if}
</div>
