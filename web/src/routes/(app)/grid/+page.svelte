<script lang="ts">
  import { onMount } from 'svelte';
  import type { Server } from '$lib/models';
	import ScreenshotGrid from '$lib/components/grid/ScreenshotGrid.svelte';

  let servers: Server[] = [];
  let filteredServers: Server[] = [];

  async function fetchServers() {
    try {
      const response = await fetch('/api/servers');
      servers = await response.json();
      filteredServers = servers;
    } catch (error) {
      console.error('Failed to fetch server data:', error);
    }
  }

  onMount(fetchServers);
</script>

<div class="p-4 min-h-screen w-full">
  <ScreenshotGrid sites={filteredServers} />
</div>

