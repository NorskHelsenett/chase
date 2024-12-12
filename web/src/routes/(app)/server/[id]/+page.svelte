<script lang="ts">
  import { onMount } from 'svelte';
  import type { Server } from '$lib/models';
  import StatusIndicator from '$lib/components/server/StatusIndicator.svelte';
  import StatusMetrics from '$lib/components/server/StatusMetrics.svelte';
  import ResponseTimeGraph from '$lib/components/server/ResponseTimeGraph.svelte';
	import ServerInfoCard from '$lib/components/server/ServerInfoCard.svelte';
	import SecurityScan from '$lib/components/SecurityScan.svelte';

  /** @type {import('./$types').PageData} */
  export let data;

  let serverID: number = 0;
  let server: Server | null = null;
  let isLoading = true;
  let isLoadingResults = true;
  let error: string | null = null;

  let searchResults = null

  $: if (data.id) {
    serverID = data.id;
  }

  onMount(() => {
    fetchServerData(serverID);
    fetchServerReport(serverID);
  });

  async function fetchServerReport(id: number) {
    try{
      const response = await fetch(`/api/servers/${id}/report`);
      if (!response.ok) throw new Error('Failed to fetch server data');

      searchResults = await response.json();
    }
    finally {
      isLoadingResults = false
    }
  }

  async function fetchServerData(id: number) {
    isLoading = true;
    error = null;

    try {
      const response = await fetch(`/api/servers/${id}`);
      if (!response.ok) throw new Error('Failed to fetch server data');

      const data: Server = await response.json();
      server = data;
    } catch (e) {
      error = e instanceof Error ? e.message : 'An error occurred';
      server = null;
    } finally {
      isLoading = false;
    }
  }

  function getLatestPing(server: Server) {
    return server.ping_results[0] || null;
  }

  function calculateMetrics(server: Server) {
    const latestPing = getLatestPing(server);
    const last24hPings = server.ping_results.filter(
      ping => new Date(ping.timestamp).getTime() > Date.now() - 24 * 60 * 60 * 1000
    );

    const avgResponse = Math.round(
      last24hPings.reduce((acc, ping) => acc + ping.response_time_ms, 0) / last24hPings.length
    );

    const uptimeDay = Math.round(
      (last24hPings.filter(ping => !ping.error && ping.status_code < 400).length / last24hPings.length) * 100
    );

    const last30DayPings = server.ping_results.filter(
      ping => new Date(ping.timestamp).getTime() > Date.now() - 30 * 24 * 60 * 60 * 1000
    );

    const uptimeMonth = Math.round(
      (last30DayPings.filter(ping => !ping.error && ping.status_code < 400).length / last30DayPings.length) * 100
    );

    return {
      currentResponse: latestPing?.response_time_ms || 0,
      avgResponse,
      uptimeDay,
      uptimeMonth,
      certDaysLeft: latestPing ?
        Math.ceil((new Date(latestPing.cert_expiry_date).getTime() - Date.now()) / (1000 * 60 * 60 * 24)) :
        0,
      certExpDate: latestPing?.cert_expiry_date
    };
  }
</script>

<div class="flex flex-col gap-4 p-4 min-h-screen">
  {#if isLoading}
    <div class="text-gray-400">Loading...</div>
  {:else if error}
    <div class="text-red-400">Error: {error}</div>
  {:else if server}

    <StatusIndicator
    pingResults={server.ping_results}
    />

    <ServerInfoCard {server} />

    <StatusMetrics
      {...calculateMetrics(server)}
    />

    <ResponseTimeGraph
      data={server.ping_results.map(ping => ({
        timestamp: new Date(ping.timestamp),
        value: ping.response_time_ms
      }))}
    />
  {/if}
  {#if isLoadingResults}
    <div class="bg-[#202020] rounded-lg p-6 animate-pulse">
      <div class="h-48 bg-gray-700 rounded-lg w-full mb-4"></div>
      <div class="space-y-2">
        {#each Array(3) as _}
          <div class="h-4 bg-gray-700 rounded w-full"></div>
        {/each}
      </div>
    </div>
  {:else}
    <SecurityScan {searchResults}/>
  {/if}
</div>