<!-- ServerControls.svelte -->
<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { Server } from '$lib/models';
	import ServerDialog from '../ServerDialog.svelte';

	const dispatch = createEventDispatcher();

	export let server: Server;
	export let isLoading = false;

	let showDialog = false;
	let dialogData: Partial<Server> | null = null;

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

	function handleDialogDelete() {
		dispatch('delete');
		showDialog = false;
	}

	function handleToggleActive(event: CustomEvent) {
		dispatch('toggleActive', { active: event.detail });
	}

	function onClose() {
		showDialog = false;
		dialogData = null;
	}
</script>

<button
	on:click={handleDialogOpen}
	disabled={isLoading}
	class="edit-btn"
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

<style>
	.edit-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 0.375rem;
		color: #9ca3af;
		background: #2b2b2b;
		border: none;
		transition: all 0.15s ease;
		cursor: pointer;
	}

	.edit-btn:hover:not(:disabled) {
		color: #e5e7eb;
		background: #333;
	}

	.edit-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}
</style>

<ServerDialog
	bind:showDialog
	{isLoading}
	mode="edit"
	initialData={dialogData}
	on:submit={handleDialogSubmit}
	on:delete={handleDialogDelete}
	on:toggleActive={handleToggleActive}
	on:close={onClose}
/>
