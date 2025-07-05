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

function normalizeUrl(url: string): string {
  return url.startsWith('http') ? url : `https://${url}`;
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
  const addedNodeIds = new Set(); // Track which node IDs have been added
  const subdomainSites = new Map(); // Track sites per subdomain

  // Helper function to add a node only if it doesn't exist yet
  const addUniqueNode = (node: GraphNode) => {
    if (!addedNodeIds.has(node.id)) {
      nodes.push(node);
      addedNodeIds.add(node.id);
      return true;
    }
    return false;
  };

  // First pass: collect all subdomain -> site relationships
  servers.forEach((server, idx) => {
    try {
      const serverUrl = normalizeUrl(server.url);
      const url = new URL(serverUrl);
      const hostname = url.hostname;
      const siteId = `site-${server.ID || idx}-${server.url}`;
      
      if (!subdomainSites.has(hostname)) {
        subdomainSites.set(hostname, []);
      }
      subdomainSites.get(hostname).push(siteId);
    } catch (error) {
      // Skip invalid URLs in this pass
    }
  });

  servers.forEach((server, idx) => {
    try {
      // Ensure the URL has a protocol
      const serverUrl = normalizeUrl(server.url);
      const url = new URL(serverUrl);
      const domain = url.hostname.split('.').slice(-2).join('.');
      const subdomain = url.hostname.replace(`.${domain}`, '');
      const hostname = url.hostname;

      // Add domain node if it doesn't exist
      if (!domainMap.has(domain)) {
        const domainServers = servers.filter(s => {
          try {
            const serverUrl = normalizeUrl(s.url);
            return new URL(serverUrl).hostname.includes(domain);
          } catch (e) {
            return false;
          }
        });

        addUniqueNode({
          id: domain,
          label: domain,
          group: 'domain',
          title: `Domain: ${domain}
                  Sites: ${domainServers.length}`
        });
        domainMap.set(domain, { subdomains: new Set() });
      }

      // Check if this subdomain has only one site
      const sitesCount = subdomainSites.get(hostname)?.length || 0;
      const skipSubdomain = subdomain && sitesCount === 1;

      // Add subdomain node if it doesn't exist and has more than one site
      if (subdomain && !skipSubdomain && !domainMap.get(domain).subdomains.has(subdomain)) {
        addUniqueNode({
          id: hostname,
          label: hostname,
          group: 'subdomain',
          title: `Subdomain: ${hostname}
                  Sites: ${sitesCount} `
        });
        edges.push({ from: domain, to: hostname });
        domainMap.get(domain).subdomains.add(subdomain);
      }

      // Add site node with unique ID by using server.ID or index as suffix if needed
      const siteId = `site-${server.ID || idx}-${server.url}`;

      // Get status info for tooltip
      const latestPing = server.ping_results && server.ping_results.length > 0
        ? server.ping_results.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())[0]
        : null;
      const statusText = !latestPing ? 'Unknown' :
                        latestPing.status_code === server.expected_status ? 'Up' : 'Down';

      addUniqueNode({
        id: siteId,
        label: server.url,
        group: 'site',
        title: `${server.url}
                Status: ${statusText}
                ${latestPing ? `Response: ${latestPing.status_code}` : ''}
                ${server.comment ? `Note: ${server.comment}` : ''} `
      });
      
      // Connect site directly to domain if subdomain is skipped, otherwise to subdomain
      if (skipSubdomain) {
        edges.push({ from: domain, to: siteId });
      } else {
        edges.push({ from: hostname, to: siteId });
      }
    } catch (error) {
      console.error(`Error processing server URL: ${server.url}`, error);
      // Still add the node even if URL parsing failed, just don't create edges
      const errorId = `error-${server.ID || idx}-${server.url}`;
      addUniqueNode({
        id: errorId,
        label: server.url,
        group: 'error',
        title: `${server.url}
                Invalid URL format
                Missing protocol (http:// or https://)?`
      });
    }
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
