<!-- ServerControls.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { Server } from '$lib/models';
  import RadioToggle from '../ui/RadioToggle.svelte';
  import ServerDialog from '../ServerDialog.svelte';

  const dispatch = createEventDispatcher();

  export let server: Server;
  export let isLoading = false;

  let showDialog = false;
  let serverActive = server?.active ?? false;
  let serverData: Server;

  $: {
    // Clone server data whenever server prop changes
    serverData = JSON.parse(JSON.stringify(server));
    serverData.active = serverActive;
  }

  // Handle keyboard shortcuts
  function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'e' && !event.ctrlKey && !event.altKey && !event.metaKey) {
      event.preventDefault();
      showDialog = true;
    }
  }

  function handleActiveChange(active: boolean) {
    serverActive = active;
    dispatch('toggleActive', { active });
  }

  function handleDelete() {
    if (confirm(`Are you sure you want to delete ${serverData.url}?`)) {
      dispatch('delete');
    }
  }

  async function handleDialogSubmit(event: CustomEvent) {
    const { data } = event.detail;

    try {
      const response = await fetch(`/api/servers/${serverData.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...data,
          id: serverData.id
        }),
      });

      if (response.ok) {
        dispatch('update', { data });
        showDialog = false;
      } else {
        const errorText = await response.text();
        console.error('Failed to update server:', errorText);
      }
    } catch (error) {
      console.error('Error updating server:', error);
    }
  }

  function onClose() {
    showDialog = false;
  }
</script>

<svelte:window on:keydown={handleKeyDown} />

<div class="bg-[#202020] rounded-lg p-4 flex items-center justify-between gap-4">
  <div class="flex items-center gap-6">
    <RadioToggle
      bind:value={serverActive}
      on:change={({ detail }) => handleActiveChange(detail)}
      label="Status"
    />

    <div class="text-gray-400 text-sm">
      Press 'e' to edit
    </div>
  </div>

  <div class="flex items-center gap-3">
    <button
      on:click={() => showDialog = true}
      disabled={isLoading}
      class="p-2 text-gray-400 hover:text-gray-200 transition-colors rounded-lg hover:bg-[#2b2b2b] disabled:opacity-50"
      title="Edit server (press 'e')"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
      </svg>
    </button>

    <button
      on:click={handleDelete}
      disabled={isLoading}
      class="p-2 text-gray-400 hover:text-red-500 transition-colors rounded-lg hover:bg-[#2b2b2b] disabled:opacity-50"
      title="Delete server"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
      </svg>
    </button>
  </div>
</div>

<ServerDialog
  bind:showDialog
  {isLoading}
  mode="edit"
  initialData={serverData}
  on:submit={handleDialogSubmit}
  on:close={onClose}
/>