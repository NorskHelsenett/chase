<!-- MonitorControls.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import ServerDialog from '../ServerDialog.svelte';
  import CustomSelect from '../ui/CustomSelect.svelte';
  import { Filter, Download } from 'lucide-svelte';

  const dispatch = createEventDispatcher();

  export let isLoading = false;

  let showDialog = false;
  let searchQuery = '';
  let filterStatus = 'all';

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

  // Handle filter change
  function handleFilterChange(event) {
    dispatch('filter', { status: event.detail.value });
  }

  function onClose() {
    showDialog = false;
  }
</script>

<div class="bg-[#202020] rounded-lg p-4 mb-4">
  <div class="flex items-center justify-between gap-4">
    <!-- Search -->
    <div class="flex-1 relative">
      <input
        type="text"
        bind:value={searchQuery}
        on:input={handleSearch}
        placeholder="Search domains..."
        class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-green-500"
      />
      {#if searchQuery}
        <button
          class="absolute right-2 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-200 transition-colors p-1"
          on:click={() => { searchQuery = ''; handleSearch(); }}
          aria-label="Clear search"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        </button>
      {/if}
    </div>

    <!-- Filter dropdown -->
    <div class="relative flex items-center z-10">
      <CustomSelect
        bind:value={filterStatus}
        icon={Filter}
        options={[
          {
            value: 'all',
            label: 'All servers',
            icon: '<div class="flex items-center"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg><span class="text-gray-100 ml-2"> Show all</span></div>'
          },
          {
            value: 'online',
            label: 'Online',
            icon: '<div class="flex items-center"><span class="w-2 h-2 bg-green-400 rounded-full mr-2 animate-pulse"></span><span class="text-green-400">Online</span></div>'
          },
          {
            value: 'issues',
            label: 'With issues',
            icon: '<div class="flex items-center"><span class="w-2 h-2 bg-red-400 rounded-full mr-2"></span><span class="text-red-400">Issues</span></div>'
          },
          {
            value: 'new',
            label: 'New',
            icon: '<div class="flex items-center"><span class="w-2 h-2 bg-gray-400 rounded-full mr-2"></span><span class="text-gray-300">New</span></div>'
          }
        ]}
        on:change={handleFilterChange}
      />
    </div>

    <!-- Control buttons -->
    <div class="flex gap-3">
      <button
        on:click={() => dispatch('export')}
        disabled={isLoading}
        class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 flex items-center gap-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        title="Export current view as CSV"
      >
        <Download class="w-4 h-4" />
      </button>

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