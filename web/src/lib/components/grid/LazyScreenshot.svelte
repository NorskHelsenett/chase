<script>
	import { run } from 'svelte/legacy';

	let imageErrors = $state({});
	let imageLoaded = $state({});

	// A cache-only request (?cached=true) 404s on a miss and kicks off a background
	// capture server-side — but that capture is capped, so a fast scroll over many
	// uncached tiles can 404 before its capture runs. Retry the miss a few times with
	// backoff so the tile fills in once the capture lands, instead of getting stuck on
	// "Failed to load". Genuine failures (server-cached error 404s) exhaust the retries
	// quickly and then show the error state.
	const maxRetries = 6;
	const retryCounts = {};
	const retryTimers = {};

	function clearRetry(url) {
		if (retryTimers[url]) {
			clearTimeout(retryTimers[url]);
			delete retryTimers[url];
		}
	}

	function checkImage(url) {
		if (imageLoaded[url] || imageErrors[url]) return;
		const img = new Image();
		img.onload = () => {
			clearRetry(url);
			imageLoaded[url] = true;
		};
		img.onerror = () => {
			const attempts = (retryCounts[url] || 0) + 1;
			retryCounts[url] = attempts;
			if (attempts <= maxRetries) {
				const delay = Math.min(3000 + attempts * 1500, 10000);
				retryTimers[url] = setTimeout(() => {
					delete retryTimers[url];
					checkImage(url);
				}, delay);
				return;
			}
			imageErrors[url] = true;
		};
		img.src = url;
	}

	function lazyCheck(node, url) {
		let currentUrl = url;
		let observer;

		function setupObserver() {
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
					rootMargin: '100px'
				}
			);

			observer.observe(node);
		}

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
				clearRetry(currentUrl);
			}
		};
	}

	let { site, getScreenshotUrl, getThumbUrl = undefined } = $props();

	// The display URL is what we actually show — thumb if available, full otherwise
	let displayUrl = $state();
	let fullUrl = $state();
	let prevDisplayUrl = $state('');

	run(() => {
		fullUrl = getScreenshotUrl(site.url);
		displayUrl = getThumbUrl ? getThumbUrl(site.url) : fullUrl;
		if (displayUrl !== prevDisplayUrl) {
			clearRetry(prevDisplayUrl);
			delete retryCounts[prevDisplayUrl];
			delete imageLoaded[prevDisplayUrl];
			delete imageErrors[prevDisplayUrl];
			prevDisplayUrl = displayUrl;
		}
	});
</script>

<!-- Container that triggers lazy load using the actual display URL -->
<div use:lazyCheck={displayUrl} class="w-full h-full">
	{#if imageLoaded[displayUrl]}
		{#if getThumbUrl}
			<img
				src={displayUrl}
				srcset="{displayUrl} 480w, {fullUrl} 1280w"
				sizes="(max-width: 768px) 100vw, (max-width: 1024px) 50vw, 25vw"
				alt={`Screenshot of ${site.url}`}
				decoding="async"
				class="absolute inset-0 w-full h-full object-cover transition-all duration-300 group-hover:scale-105 group-hover:brightness-105 rounded-t-xl"
			/>
		{:else}
			<img
				src={displayUrl}
				alt={`Screenshot of ${site.url}`}
				decoding="async"
				class="absolute inset-0 w-full h-full object-cover transition-all duration-300 group-hover:scale-105 group-hover:brightness-105 rounded-t-xl"
			/>
		{/if}
	{:else if imageErrors[displayUrl]}
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
