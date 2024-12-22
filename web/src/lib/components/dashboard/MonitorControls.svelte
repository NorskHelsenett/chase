<!-- MonitorControls.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import ServerDialog from '../ServerDialog.svelte';

  const dispatch = createEventDispatcher();

  export let isLoading = false;

  let showDialog = false;
  let searchQuery = '';

  // Handle dialog submission
  async function handleDialogSubmit(event: CustomEvent) {
    const { data, mode } = event.detail;

    try {
      const response = await fetch('/api/servers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });

      if (response.ok) {
        dispatch('serverAdded');
        showDialog = false;
        handleRefresh();
      } else {
        console.error('Failed to add server:', await response.text());
      }
    } catch (error) {
      console.error('Error adding server:', error);
    }
  }

  // Handle search
  function handleSearch() {
    dispatch('search', { query: searchQuery });
  }

  // Handle refresh
  function handleRefresh() {
    dispatch('refresh');
  }

  function onClose() {
    showDialog = false;
  }
</script>

<div class="bg-[#202020] rounded-lg p-4 mb-4">
  <div class="flex items-center justify-between gap-4">
    <!-- Search -->
    <div class="flex-1">
      <input
        type="text"
        bind:value={searchQuery}
        on:input={handleSearch}
        placeholder="Search domains..."
        class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-green-500"
      />
    </div>

    <!-- Control buttons -->
    <div class="flex gap-3">
      <button
        on:click={handleRefresh}
        disabled={isLoading}
        class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 flex items-center gap-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <svg
          class="w-4 h-4 {isLoading ? 'animate-spin' : ''}"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          />
        </svg>
        {isLoading ? 'Refreshing...' : 'Refresh'}
      </button>

      <button
        on:click={() => showDialog = true}
        disabled={isLoading}
        class="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white flex items-center gap-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Add Server
      </button>
    </div>
  </div>
</div>

<ServerDialog
  bind:showDialog
  {isLoading}
  mode="add"
  initialData={null}
  on:submit={handleDialogSubmit}
  on:close={onClose}
/>