<script lang="ts">
  import { page } from '$app/stores';
  import { derived } from 'svelte/store';
  import { onMount } from 'svelte';
  import MonitorStats from "$lib/components/dashboard/MonitorStats.svelte";
  import MonitorControls from "$lib/components/dashboard/MonitorControls.svelte";
  import MonitorTable from "$lib/components/dashboard/MonitorTable.svelte";
  import { servers, isLoading, serverStats, serverStoreActions } from '$lib/stores/serverStore';
  import type { Server } from '$lib/models';

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

  // Watch for changes in activeFilter and refetch data
  $: if (activeFilter !== undefined) {
    fetchServers();
  }

  onMount(async () => {
    // Initial fetch from cache or API (server data with ping_results included)
    await fetchServers(true);
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