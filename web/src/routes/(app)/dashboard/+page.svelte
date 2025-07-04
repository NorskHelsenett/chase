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
  let visibleServerIds = new Set<string>();
  let observer: IntersectionObserver;

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

  // Update visible servers when the filtered list changes
  $: if (filteredServers) {
    updateVisibleServers();
  }

  async function fetchServers(forceRefresh = false) {
    await serverStoreActions.loadServers(activeFilter, forceRefresh);
  }

  function handleSearch(event: CustomEvent) {
    searchQuery = event.detail.query.toLowerCase();
  }

  function handleRefresh() {
    fetchServers(true); // Force refresh from server
  }

  // Set up the intersection observer to detect which servers are in view
  function setupObserver() {
    if (typeof IntersectionObserver === 'undefined') {
      // Fallback for browsers that don't support IntersectionObserver
      console.warn('IntersectionObserver not supported, loading all server details');
      return;
    }

    observer = new IntersectionObserver(
      (entries) => {
        entries.forEach(entry => {
          const id = entry.target.getAttribute('data-server-id');
          if (id) {
            if (entry.isIntersecting) {
              visibleServerIds.add(id);
              // Security data now comes from server endpoint, no need to load separately
            } else {
              visibleServerIds.delete(id);
            }
          }
        });
      },
      { rootMargin: '100px 0px' }
    );
  }

  // This function is no longer needed as security data comes with the server data
  async function loadServerDetail(serverId: string) {
    // No longer need to load security data separately
    // Security data is now included in the main server response
    return;
  }

  // Update which servers are considered "visible" and observed
  function updateVisibleServers() {
    if (!observer) return;

    // Disconnect any previous observations
    observer.disconnect();

    // Start observing all server rows
    setTimeout(() => {
      document.querySelectorAll('[data-server-id]').forEach(element => {
        observer.observe(element);
      });
    }, 100);
  }

  // Watch for changes in activeFilter and refetch data
  $: if (activeFilter !== undefined) {
    fetchServers();
  }

  onMount(async () => {
    // Set up the intersection observer
    setupObserver();

    // Initial fetch from cache or API (server data with ping_results included)
    await fetchServers();

    // After rendering, start observing which servers are visible
    updateVisibleServers();
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
    <MonitorTable sites={filteredServers} bind:visibleServerIds/>
  {/if}
</div>