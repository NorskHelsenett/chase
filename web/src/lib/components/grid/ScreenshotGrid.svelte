<script lang="ts">
	import type { Server } from '$lib/models';
	import ScreenshotModal from './ScreenshotModal.svelte';
	import LazyScreenshot from './LazyScreenshot.svelte';
	import { fade, scale } from 'svelte/transition';
	import { getEffectiveStatus } from '$lib/utils/status';
	interface Props {
		sites?: Server[];
	}

	let { sites = [] }: Props = $props();
	let selectedImageIndex: number | null = $state(null);

	function getScreenshotUrl(url: string) {
		const cleanUrl = url.replace(/^(https?:\/\/)/, '').replace(/\/$/, '');
		return `/api/screenshot/${cleanUrl}?cached=true`;
	}

	function getThumbUrl(url: string) {
		const cleanUrl = url.replace(/^(https?:\/\/)/, '').replace(/\/$/, '');
		return `/api/screenshot/${cleanUrl}?cached=true&thumb=true`;
	}

	function openModal(index: number) {
		selectedImageIndex = index;
	}

	function closeModal() {
		selectedImageIndex = null;
	}

	function handleClick(event: MouseEvent, site: Server, index: number) {
		// Check if cmd (Mac) or ctrl (Windows/Linux) key is pressed
		if (event.metaKey || event.ctrlKey) {
			openSiteUrl(site.url);
		} else {
			// Normal click behavior - open the modal
			openModal(index);
		}
	}

	function handleKeyDown(event: KeyboardEvent, site: Server, index: number) {
		// Enter key opens modal, Enter+Ctrl/Cmd opens URL
		if (event.key === 'Enter') {
			if (event.metaKey || event.ctrlKey) {
				openSiteUrl(site.url);
			} else {
				openModal(index);
			}
		}
	}

	function openSiteUrl(url: string) {
		// Ensure the URL has a protocol
		if (!url.startsWith('http://') && !url.startsWith('https://')) {
			url = 'https://' + url;
		}

		// Open the URL in a new tab
		window.open(url, '_blank', 'noopener,noreferrer');
	}

	function getHostname(url: string): string {
		try {
			return new URL(url.startsWith('http') ? url : `https://${url}`).hostname;
		} catch {
			return url;
		}
	}
</script>

<div class="bg-[#202020] rounded-xl p-5 shadow-lg border border-green-900/30">
	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
		{#each sites as site, index}
			{#if site.active}
				<div
					in:fade={{ duration: 300, delay: index * 70 }}
					class="relative group rounded-xl transition-all duration-300 overflow-hidden cursor-pointer bg-gradient-to-br from-[#1a1a1a] to-[#222] shadow-md hover:border-green-500/50 hover:shadow-xl hover:shadow-green-900/5 hover:ring-2 hover:ring-green-500"
					onclick={(e) => handleClick(e, site, index)}
					onkeydown={(e) => handleKeyDown(e, site, index)}
					title="Click to view, Cmd/Ctrl+Click to open website"
					tabindex="0"
					role="button"
					aria-label="Screenshot of {site.url}. Click to view larger, Cmd or Ctrl click to visit website."
				>
					<div class="relative w-full pb-[56.25%] overflow-hidden">
						<LazyScreenshot {site} {getScreenshotUrl} {getThumbUrl} />

						<!-- Status badge -->
						<div
							class="absolute top-3 right-3 transition-transform duration-300 group-hover:scale-105"
						>
							{#if getEffectiveStatus(site) === 'up'}
								<span
									class="bg-green-500/30 text-green-400 text-xs px-2.5 py-1 rounded-full flex items-center gap-1.5 shadow-lg backdrop-blur-sm border border-green-500/20"
								>
									<span class="w-1.5 h-1.5 bg-green-400 rounded-full animate-pulse"></span>
									<span class="font-medium">Online</span>
								</span>
							{:else if getEffectiveStatus(site) === 'down'}
								<span
									class="bg-red-500/30 text-red-400 text-xs px-2.5 py-1 rounded-full flex items-center gap-1.5 shadow-lg backdrop-blur-sm border border-red-500/20"
								>
									<span class="w-1.5 h-1.5 bg-red-400 rounded-full animate-ping"></span>
									<span class="font-medium">Issues</span>
								</span>
							{:else}
								<span
									class="bg-gray-500/30 text-gray-300 text-xs px-2.5 py-1 rounded-full flex items-center gap-1.5 shadow-lg backdrop-blur-sm border border-gray-500/20"
								>
									<span class="w-1.5 h-1.5 bg-gray-300 rounded-full"></span>
									<span class="font-medium">New</span>
								</span>
							{/if}
						</div>

						<div
							class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/90 to-black/40 backdrop-blur-sm p-3.5 transform translate-y-full transition-transform duration-300 group-hover:translate-y-0 rounded-b-xl"
						>
							<div class="flex justify-between items-center gap-2">
								<div class="flex-1 min-w-0">
									<p class="text-white text-sm font-medium truncate">
										{getHostname(site.url)}
									</p>
									<p class="text-xs text-gray-300 truncate">{site.url}</p>
								</div>
								<span
									class="bg-green-500/30 text-green-300 text-xs px-2.5 py-1 rounded-lg border border-green-500/20 font-medium shadow-sm"
								>
									View
								</span>
							</div>
						</div>
					</div>
				</div>
			{/if}
		{:else}
			<div class="col-span-full py-16 text-center">
				<div
					in:scale={{ duration: 400 }}
					class="mx-auto mb-5 w-16 h-16 rounded-full bg-green-500/10 flex items-center justify-center"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="32"
						height="32"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="1.5"
						stroke-linecap="round"
						stroke-linejoin="round"
						class="text-green-500/70"
					>
						<rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
						<line x1="8" y1="21" x2="16" y2="21"></line>
						<line x1="12" y1="17" x2="12" y2="21"></line>
					</svg>
				</div>
				<p in:fade={{ duration: 300, delay: 100 }} class="text-lg font-medium text-green-400 mb-2">
					No servers found
				</p>
				<p in:fade={{ duration: 300, delay: 200 }} class="text-sm text-gray-400 max-w-md mx-auto">
					Try adjusting your search filters or add a new server to monitor
				</p>
			</div>
		{/each}
	</div>

	{#if selectedImageIndex !== null}
		<ScreenshotModal {sites} currentIndex={selectedImageIndex} onClose={closeModal} />
	{/if}
</div>
