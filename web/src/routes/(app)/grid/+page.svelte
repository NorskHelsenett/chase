<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import type { Server } from '$lib/models';
  import ScreenshotGrid from '$lib/components/grid/ScreenshotGrid.svelte';
  import { Search, Grid, Server as ServerIcon, Filter } from 'lucide-svelte';

  let servers: Server[] = [];
  let filteredServers: Server[] = [];
  let loading = true;
  let searchTerm = '';
  let filterStatus = 'all'; // 'all', 'online', 'issues', 'new'

  function isSuccessfulStatus(status: number): boolean {
    return status >= 200 && status < 400;
  }

  function hasGoodPingHistory(server: Server): boolean {
    if (server.ping_results.length === 0) {
      return true; // New server with no pings
    }

    // Calculate success rate of all pings
    const successfulPings = server.ping_results.filter(ping =>
      isSuccessfulStatus(ping.status_code)
    ).length;

    const successRate = successfulPings / server.ping_results.length;
    return successRate >= 0.9; // 90% success rate threshold
  }

  $: activeFilter = $page.url.searchParams.get('active');

  $: {
    if (servers && servers.length > 0) {
      // First filter by active status
      let result = servers.filter((server: Server) => server.active);

      // Then filter by search term if it exists
      if (searchTerm) {
        const term = searchTerm.toLowerCase();
        result = result.filter((server: Server) =>
          server.url.toLowerCase().includes(term) ||
          server.name?.toLowerCase().includes(term) ||
          server.description?.toLowerCase().includes(term)
        );
      }

      // Then apply status filter
      if (filterStatus !== 'all') {
        if (filterStatus === 'online') {
          result = result.filter((server: Server) =>
            server.ping_results.length > 0 && hasGoodPingHistory(server)
          );
        } else if (filterStatus === 'issues') {
          result = result.filter((server: Server) =>
            server.ping_results.length > 0 && !hasGoodPingHistory(server)
          );
        } else if (filterStatus === 'new') {
          const thirtyDaysAgo = new Date();
          thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
          result = result.filter((server: Server) => new Date(server.CreatedAt) >= thirtyDaysAgo);
        }
      }

      filteredServers = result;
    }
  }

  async function fetchServers() {
    loading = true;
    try {
      const url = new URL('/api/servers', window.location.origin);
      if (activeFilter !== null) {
        url.searchParams.set('active', activeFilter);
      }

      const response = await fetch(url);
      servers = await response.json();

      // Initial filtering happens in the reactive statement above
    } catch (error) {
      console.error('Failed to fetch server data:', error);
    } finally {
      loading = false;
    }
  }

  onMount(fetchServers);
</script>

<div class="p-4 min-h-screen w-full">
  <!-- Header and filters -->
  <div class="mb-6 flex flex-col gap-4">
    <div class="flex flex-wrap justify-between items-center gap-4">
      <h1 class="text-2xl font-medium flex items-center gap-2">
        <Grid size={24} class="text-green-500"/>
        Server Grid View
        <span class="text-sm font-normal bg-green-500/20 text-green-300 px-3 py-1 rounded-full ml-2 shadow-inner">
          {filteredServers.length} {filteredServers.length === 1 ? 'server' : 'servers'}
        </span>
      </h1>

      <div class="flex flex-wrap items-center gap-3">
        <div class="relative group">
          <Search size={18} class="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-500 group-focus-within:text-green-500 transition-colors" />
          <input
            type="text"
            bind:value={searchTerm}
            placeholder="Search servers..."
            class="pl-10 pr-4 py-2.5 bg-black/30 border border-green-900/30 rounded-xl focus:outline-none focus:ring-2 focus:ring-green-500/70 focus:border-transparent transition-all"
          />
        </div>

        <div class="relative flex items-center">
          <Filter size={18} class="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-500" />
          <select
            bind:value={filterStatus}
            class="appearance-none pl-10 pr-10 py-2.5 bg-black/30 border border-green-900/30 rounded-xl focus:outline-none focus:ring-2 focus:ring-green-500/70 focus:border-transparent transition-all"
          >
            <option value="all">All servers</option>
            <option value="online">Online</option>
            <option value="issues">With issues</option>
            <option value="new">New</option>
          </select>
          <div class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-gray-500">
            <svg class="fill-current h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20">
              <path d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" />
            </svg>
          </div>
        </div>
      </div>
    </div>
  </div>

  {#if loading}
    <div class="flex flex-col items-center justify-center py-16">
      <div class="relative">
        <div class="w-16 h-16 border-4 border-t-green-500 border-r-green-400/40 border-b-green-400/20 border-l-green-400/60 rounded-full animate-spin"></div>
        <div class="absolute inset-0 w-16 h-16 border-4 border-green-500/10 rounded-full animate-pulse"></div>
      </div>
      <p class="mt-5 text-green-400 font-medium">Loading servers...</p>
    </div>
  {:else}
    <ScreenshotGrid sites={filteredServers} />
  {/if}
</div>