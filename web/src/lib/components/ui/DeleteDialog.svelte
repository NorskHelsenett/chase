<!-- DeleteDialog.svelte -->
<script lang="ts">
	import { fade } from 'svelte/transition';
	import { createEventDispatcher } from 'svelte';
	import type { Server } from '$lib/models';
	import { Trash2, Loader2 } from 'lucide-svelte';

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
	<div class="overlay" transition:fade>
		<div class="dialog">
			<div class="dialog-header">
				<div class="icon-wrapper">
					<Trash2 size={24} />
				</div>
				<h2>{title}</h2>
			</div>

			<div class="dialog-body">
				<p class="message">
					Are you sure you want to delete <span class="highlight">{serverData.url}</span>?
				</p>
				<p class="warning">
					This action cannot be undone. All monitoring data, history, and settings for this server
					will be permanently removed.
				</p>
			</div>

			<div class="dialog-footer">
				<button
					type="button"
					on:click={handleClose}
					disabled={isLoading}
					class="btn btn-secondary"
				>
					Cancel
				</button>
				<button
					type="button"
					on:click={handleSubmit}
					disabled={isLoading}
					class="btn btn-danger"
				>
					{#if isLoading}
						<Loader2 size={16} class="spinning" />
					{/if}
					<span>{isLoading ? loadingLabel : submitLabel}</span>
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 50;
	}

	.dialog {
		background: #202020;
		border-radius: 0.5rem;
		padding: 1.5rem;
		width: 100%;
		max-width: 32rem;
		margin: 1rem;
	}

	.dialog-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 1.5rem;
	}

	.icon-wrapper {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 3rem;
		height: 3rem;
		border-radius: 50%;
		background: rgba(239, 68, 68, 0.1);
		color: #ef4444;
	}

	.dialog-header h2 {
		font-size: 1.25rem;
		font-weight: 600;
		color: #e5e7eb;
	}

	.dialog-body {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.message {
		color: #d1d5db;
	}

	.highlight {
		font-weight: 600;
		color: #fff;
	}

	.warning {
		font-size: 0.875rem;
		color: #9ca3af;
	}

	.dialog-footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: 2rem;
	}

	.btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		border: none;
		border-radius: 0.5rem;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s ease, opacity 0.15s ease;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-secondary {
		background: #2b2b2b;
		color: #e5e7eb;
	}

	.btn-secondary:hover:not(:disabled) {
		background: #333;
	}

	.btn-danger {
		background: #dc2626;
		color: white;
	}

	.btn-danger:hover:not(:disabled) {
		background: #b91c1c;
	}

	:global(.spinning) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
