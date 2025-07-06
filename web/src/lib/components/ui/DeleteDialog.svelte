<!-- DeleteDialog.svelte -->
<script lang="ts">
	import { fade } from 'svelte/transition';
	import { createEventDispatcher } from 'svelte';
	import type { Server } from '$lib/models';

	const dispatch = createEventDispatcher();

	export let showDialog = false;
	export let isLoading = false;
	export let serverData: Server;

	let title = 'Delete Server';
	let submitLabel = 'Delete Server';
	let loadingLabel = 'Deleting...';

	function handleSubmit() {
		dispatch('submit');
	}

	function handleClose() {
		showDialog = false;
		dispatch('close');
	}
</script>

{#if showDialog}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" transition:fade>
		<div class="bg-[#202020] rounded-lg p-6 w-full max-w-lg">
			<div class="flex items-center justify-between mb-6">
				<div class="flex items-center gap-3">
					<div class="bg-red-500/10 p-3 rounded-full">
						<svg class="w-6 h-6 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
							/>
						</svg>
					</div>
					<h2 class="text-xl text-gray-200 font-semibold">{title}</h2>
				</div>
			</div>

			<div class="space-y-4">
				<p class="text-gray-300">
					Are you sure you want to delete <span class="font-semibold text-white"
						>{serverData.url}</span
					>?
				</p>
				<p class="text-gray-400 text-sm">
					This action cannot be undone. All monitoring data, history, and settings for this server
					will be permanently removed.
				</p>
			</div>

			<div class="flex justify-end gap-3 mt-8">
				<button
					type="button"
					on:click={handleClose}
					disabled={isLoading}
					class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Cancel
				</button>
				<button
					type="button"
					on:click={handleSubmit}
					disabled={isLoading}
					class="px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
				>
					{#if isLoading}
						<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
								fill="none"
							/>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							/>
						</svg>
					{/if}
					{isLoading ? loadingLabel : submitLabel}
				</button>
			</div>
		</div>
	</div>
{/if}
