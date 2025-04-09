<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import MonitorStats from "$lib/components/dashboard/MonitorStats.svelte";
  import MonitorControls from "$lib/components/dashboard/MonitorControls.svelte";
  import MonitorTable from "$lib/components/dashboard/MonitorTable.svelte";
  import { servers, isLoading, serverStats, serverStoreActions } from '$lib/stores/serverStore';
  import type { Server } from '$lib/models';
  import { derived } from 'svelte/store';

  let filteredServers: Server[] = [];
  let searchQuery = '';
  
  // Subscribe to page store to get URL parameters
  $: activeFilter = $page.url.searchParams.get('active');
  
  // Create a derived store that filters servers based on search query
  $: filteredStore = derived(servers, $servers => {
    if (!searchQuery) return $servers;
    
    const query = searchQuery.toLowerCase();
    return $servers.filter(server =>
      server.url.toLowerCase().includes(query) ||
      server.comment?.toLowerCase().includes(query)
    );
  });
  
  // Subscribe to the filtered store
  $: filteredServers = $filteredStore;

  async function fetchServers(forceRefresh = false) {
    await serverStoreActions.loadServers(activeFilter, forceRefresh);
  }

  function handleSearch(event: CustomEvent) {
    searchQuery = event.detail.query.toLowerCase();
  }

  function handleRefresh() {
    fetchServers(true); // Force refresh from server
  }

  // Load individual server details in the background
  async function loadServerDetails() {
    if ($servers && $servers.length > 0) {
      // Process in batches to avoid overwhelming the server
      const batchSize = 3;
      for (let i = 0; i < $servers.length; i += batchSize) {
        const batch = $servers.slice(i, i + batchSize);
        await Promise.all(batch.map(server => 
          serverStoreActions.loadServerPings(server.ID)
        ));
        // Small delay between batches
        if (i + batchSize < $servers.length) {
          await new Promise(r => setTimeout(r, 300));
        }
      }
    }
  }

  // Watch for changes in activeFilter and refetch data
  $: if (activeFilter !== undefined) {
    fetchServers();
  }

  onMount(async () => {
    // Initial fetch from cache or API
    await fetchServers();
    
    // Load detailed data in the background
    loadServerDetails();
  });
</script>

<div class="p-4 w-full">
  <MonitorStats stats={$serverStats} />
  <MonitorControls
    isLoading={$isLoading}
    on:search={handleSearch}
    on:refresh={handleRefresh}
    on:serverAdded={() => fetchServers(true)}
  />
  {#if $isLoading && filteredServers.length === 0}
    <div class="flex justify-center items-center p-6">
      <div class="animate-pulse text-gray-500">Loading server data...</div>
    </div>
  {:else}
    <MonitorTable sites={filteredServers}/>
  {/if}
</div>