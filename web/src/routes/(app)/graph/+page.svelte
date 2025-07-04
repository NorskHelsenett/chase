<script lang="ts">
import { onMount } from 'svelte';
import { servers, serverStoreActions } from '$lib/stores/serverStore';
import { writable } from 'svelte/store';
import type { Server } from '$lib/models';
import Graph from '$lib/components/dashboard/Graph.svelte';

const graphData = writable<{ nodes: GraphNode[]; edges: GraphEdge[] }>({ nodes: [], edges: [] });
const isLoading = writable(true);

interface GraphNode {
  id: string;
  label: string;
  group: string;
}
interface GraphEdge {
  from: string;
  to: string;
}

async function loadServersFromCacheOrFetch() {
  let cached = localStorage.getItem('servers');
  let serverList: Server[] = [];
  if (cached) {
    try {
      serverList = JSON.parse(cached);
    } catch (e) {
      serverList = [];
    }
  }
  if (!serverList.length) {
    await serverStoreActions.loadServers();
    serverList = $servers;
    localStorage.setItem('servers', JSON.stringify(serverList));
  }
  return serverList;
}

function buildGraphData(servers: Server[]): { nodes: GraphNode[]; edges: GraphEdge[] } {
  const nodes: GraphNode[] = [];
  const edges: GraphEdge[] = [];
  const domainMap = new Map();
  servers.forEach((server, idx) => {
    const url = new URL(server.url);
    const domain = url.hostname.split('.').slice(-2).join('.');
    const subdomain = url.hostname.replace(`.${domain}`, '');
    if (!domainMap.has(domain)) {
      nodes.push({ id: domain, label: domain, group: 'domain' });
      domainMap.set(domain, { subdomains: new Set() });
    }
    if (subdomain && !domainMap.get(domain).subdomains.has(subdomain)) {
      nodes.push({ id: url.hostname, label: url.hostname, group: 'subdomain' });
      edges.push({ from: domain, to: url.hostname });
      domainMap.get(domain).subdomains.add(subdomain);
    }
    nodes.push({ id: server.url, label: server.url, group: 'site' });
    edges.push({ from: url.hostname, to: server.url });
  });
  return { nodes, edges };
}

onMount(async () => {
  isLoading.set(true);
  const serverList = await loadServersFromCacheOrFetch();
  graphData.set(buildGraphData(serverList));
  isLoading.set(false);
});
</script>

<div class="p-4 w-full h-full">
  <h2 class="text-2xl font-bold mb-4">Site Graph</h2>
  {#if $isLoading}
    <div class="flex justify-center items-center p-6">
      <div class="animate-pulse text-gray-500">Loading graph data...</div>
    </div>
  {:else}
    <Graph {graphData} />
  {/if}
</div>
