<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import type { Server } from '$lib/models';
  import ScreenshotGrid from '$lib/components/grid/ScreenshotGrid.svelte';

  let servers: Server[] = [];
  let filteredServers: Server[] = [];

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

  async function fetchServers() {
    try {
      const url = new URL('/api/servers', window.location.origin);
      if (activeFilter !== null) {
        url.searchParams.set('active', activeFilter);
      }

      const response = await fetch(url);
      servers = await response.json();
      
      // Filter for active servers with good ping history
      filteredServers = servers.filter(server => 
        server.active && hasGoodPingHistory(server)
      );
    } catch (error) {
      console.error('Failed to fetch server data:', error);
    }
  }

  onMount(fetchServers);
</script>

<div class="p-4 min-h-screen w-full">
  <ScreenshotGrid sites={filteredServers} />
</div>