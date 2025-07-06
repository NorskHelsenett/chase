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

<div class="bg-[#202020] rounded-lg p-4 flex items-center justify-between gap-4">
	<div class="flex items-center gap-6">
		<ToggleButton
			bind:value={serverActive}
			onLabel="Active"
			offLabel="Inactive"
			on:change={handleActiveChange}
		/>
	</div>

	<div class="flex items-center gap-3">
		<button
			on:click={handleDialogOpen}
			disabled={isLoading}
			class="p-2 text-gray-400 hover:text-gray-200 transition-colors rounded-lg hover:bg-[#2b2b2b] disabled:opacity-50"
			title="Edit server"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
			class="p-2 text-gray-400 hover:text-red-500 transition-colors rounded-lg hover:bg-[#2b2b2b] disabled:opacity-50"
			title="Delete server"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
