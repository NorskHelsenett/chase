<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import MonitorStats from "$lib/components/dashboard/MonitorStats.svelte";
  import MonitorControls from "$lib/components/dashboard/MonitorControls.svelte";
  import MonitorTable from "$lib/components/dashboard/MonitorTable.svelte";
  import type { Server, Stats } from '$lib/models';

  let servers: Server[] = [];
  let filteredServers: Server[] = [];
  let isLoading = false;
  let stats: Stats = {
    up: 0,
    down: 0,
    criticalRisks: 0,
    highRisks: 0
  };

  // Subscribe to page store to get URL parameters
  $: activeFilter = $page.url.searchParams.get('active');

  async function fetchServers() {
    if(isLoading) {
      return
    }
    isLoading = true;
    try {

      // Build URL with query parameters
      const url = new URL('/api/servers', window.location.origin);
      if (activeFilter !== null) {
        url.searchParams.set('active', activeFilter);
      }

      const response = await fetch(url);
      servers = await response.json();
      filteredServers = servers;
      updateStats();
    } catch (error) {
      console.error('Failed to fetch server data:', error);
    } finally {
      isLoading = false;
    }
  }

  function updateStats() {
    stats = servers.reduce((acc: Stats, server: Server) => {
      const sortedPings = [...server.ping_results].sort((a, b) =>
        new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
      );
      const latestPing = sortedPings[0];

      if (latestPing) {
        if (latestPing.status_code === server.expected_status) {
          acc.up += 1;
        } else {
          acc.down += 1;
        }

        if (!latestPing.tls_valid) {
          acc.criticalRisks += 1;
        }

        const certExpiryDate = new Date(latestPing.cert_expiry_date);
        const daysUntilExpiry = Math.floor(
          (certExpiryDate.getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24)
        );

        if (daysUntilExpiry < 30 && daysUntilExpiry > 0) {
          acc.highRisks += 1;
        }
      } else {
        acc.down += 1;
      }

      return acc;
    }, {
      up: 0,
      down: 0,
      criticalRisks: 0,
      highRisks: 0
    });
  }

  function handleSearch(event: CustomEvent) {
    const query = event.detail.query.toLowerCase();
    filteredServers = servers.filter(server =>
      server.url.toLowerCase().includes(query) ||
      server.comment?.toLowerCase().includes(query)
    );
  }

  // Watch for changes in activeFilter and refetch data
  $: if (activeFilter !== undefined) {
    fetchServers();
  }

  onMount(fetchServers);
</script>

<div class="p-4 w-full">
  <MonitorStats {stats} />
  <MonitorControls
    {isLoading}
    on:search={handleSearch}
    on:refresh={fetchServers}
    on:serverAdded={fetchServers}
  />
  <MonitorTable sites={filteredServers}/>
</div>