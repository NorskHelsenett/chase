<!-- ServerControls.svelte -->
<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { Server } from '$lib/models';
	import ToggleButton from '../ui/ToggleButton.svelte';
	import ServerDialog from '../ServerDialog.svelte';
	import DeleteDialog from '../ui/DeleteDialog.svelte';

	const dispatch = createEventDispatcher();

	export let server: Server;
	export let isLoading = false;

	let showDialog = false;
	let serverActive = server?.active ?? false;
	let dialogData: Partial<Server> | null = null;
	let showDeleteDialog = false;

	function handleActiveChange(event: CustomEvent) {
		const active = event.detail;
		serverActive = active;
		dispatch('toggleActive', { active });
	}

	function handleDelete() {
		showDeleteDialog = true;
	}

	function handleDeleteConfirm() {
		dispatch('delete');
		showDeleteDialog = false;
	}

	function handleDialogOpen() {
		// Create a deep copy of the server data when opening the dialog
		dialogData = {
			id: server.id,
			url: server.url,
			active: server.active,
			follow_redirect: server.follow_redirect,
			allow_insecure: server.allow_insecure,
			expected_status: server.expected_status,
			comment: server.comment,
			update_interval: server.update_interval
		};
		showDialog = true;
	}

	function handleDialogSubmit(event: CustomEvent) {
		const { data } = event.detail;
		dispatch('update', { data });
		showDialog = false;
	}

	function onClose() {
		showDialog = false;
		dialogData = null;
	}
</script>

<div class="controls-bar">
	<ToggleButton
		bind:value={serverActive}
		onLabel="Active"
		offLabel="Paused"
		on:change={handleActiveChange}
	/>

	<div class="controls-actions">
		<button
			on:click={handleDialogOpen}
			disabled={isLoading}
			class="action-btn"
			title="Edit server"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
				/>
			</svg>
		</button>

		<button
			on:click={handleDelete}
			disabled={isLoading}
			class="action-btn delete"
			title="Delete server"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
				/>
			</svg>
		</button>
	</div>
</div>

<style>
	.controls-bar {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.controls-actions {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 0.5rem;
		color: #6b7280;
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid transparent;
		transition: all 0.15s ease;
	}

	.action-btn:hover:not(:disabled) {
		color: #d1d5db;
		background: rgba(255, 255, 255, 0.08);
		border-color: #333;
	}

	.action-btn.delete:hover:not(:disabled) {
		color: #f87171;
		background: rgba(248, 113, 113, 0.1);
		border-color: rgba(248, 113, 113, 0.2);
	}

	.action-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}
</style>

<DeleteDialog
	bind:showDialog={showDeleteDialog}
	{isLoading}
	serverData={server}
	on:submit={handleDeleteConfirm}
	on:close={() => (showDeleteDialog = false)}
/>

<ServerDialog
	bind:showDialog
	{isLoading}
	mode="edit"
	initialData={dialogData}
	on:submit={handleDialogSubmit}
	on:close={onClose}
/>
