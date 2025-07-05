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
  title?: string;
}
interface GraphEdge {
  from: string;
  to: string;
}

function normalizeUrl(url: string): string {
  return url.startsWith('http') ? url : `https://${url}`;
}

function getRootDomain(hostname: string): string {
  // Handles multi-part TLDs like .co.uk, .nhn.no etc. Could be improved with a public suffix list.
  const parts = hostname.split('.');
  if (parts.length >= 3 && parts[parts.length - 2].length <= 3 && parts[parts.length - 1].length <= 3) {
    // E.g. sikkerhet.nhn.no => nhn.no, grimsgaard.co.uk => co.uk
    return parts.slice(-3).join('.');
  }
  return parts.slice(-2).join('.');
}

function getAllDomainLevels(hostname: string): string[] {
  // Returns all levels for grouping: foo.bar.baz.com => [foo.bar.baz.com, bar.baz.com, baz.com]
  const parts = hostname.split('.');
  const levels: string[] = [];
  for (let i = 0; i < parts.length - 1; i++) {
    levels.push(parts.slice(i).join('.'));
  }
  return levels;
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

// --- Graph util (core logic for buildGraphData) ---
function buildGraphData(servers) {
  const nodes = [];
  const edges = [];

  const addedNodeIds = new Set();
  const groupHostnames = new Set();
  // Removed unused parentMap declaration

  // 1. Count all groupings
  const levelCounts = new Map();
  servers.forEach(server => {
    try {
      const hostname = new URL(normalizeUrl(server.url)).hostname;
      getAllDomainLevels(hostname).forEach(level => {
        levelCounts.set(level, (levelCounts.get(level) || 0) + 1);
      });
    } catch {}
  });

  function addNode(id, label, group, title, isDown = false) {
    if (!addedNodeIds.has(id)) {
      nodes.push({ id, label, group, title, isDown });
      addedNodeIds.add(id);
    }
  }

  // 2. Add domain/group nodes first and track hostnames
  for (const [level, count] of levelCounts) {
    if (count > 1) {
      const nodeId = `domain:${level}`;
      addNode(nodeId, level, 'domain', `Domain/group: ${level}\nCount: ${count}`);
      groupHostnames.add(level);
    }
  }

  // 3. Connect domain/group nodes to parent group, if one exists (for hierarchy)
  //    (Fixes nhn.no disconnect from sikkerhet.nhn.no)
  for (const [level, count] of levelCounts) {
    if (count > 1) {
      const parentLevels = getAllDomainLevels(level).slice(1); // strip self, parent chain
      for (const parent of parentLevels) {
        if (groupHostnames.has(parent)) {
          edges.push({ from: `domain:${parent}`, to: `domain:${level}` });
          break; // Only connect to the closest parent group
        }
      }
    }
  }

// 4. Add site and instance nodes, never adding site nodes that match groupHostnames
servers.forEach((server, idx) => {
  try {
    const serverUrl = normalizeUrl(server.url);
    const url = new URL(serverUrl);
    const hostname = url.hostname;
    const allLevels = getAllDomainLevels(hostname);
    const rootDomain = getRootDomain(hostname);

    // Find the closest parent domain/group node (deepest level first)
    let parentForInstance = null;
    for (let i = 0; i < allLevels.length; i++) {
      if (groupHostnames.has(allLevels[i])) {
        parentForInstance = `domain:${allLevels[i]}`;
        break;
      }
    }

    if (!parentForInstance) {
      // fallback to rootDomain, or create site node
      if (groupHostnames.has(rootDomain)) {
        parentForInstance = `domain:${rootDomain}`;
      } else {
        const siteNodeId = `site:${hostname}`;
        const latestPing = server.ping_results && server.ping_results.length > 0
          ? server.ping_results.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())[0]
          : null;
        // Calculate isDown for backwards compatibility
        const isDown = !latestPing || latestPing.error || latestPing.status_code !== server.expected_status;
        // Generate statusText same way as for instances
        const statusText = !latestPing ? 'Unknown' : (isDown ? 'Down' : 'Up');
        // Use statusText to determine up/down state
        const nodeIsDown = statusText.toLowerCase() === 'down';

        addNode(siteNodeId, hostname, nodeIsDown ? 'down' : 'up', server.url, nodeIsDown);
        parentForInstance = siteNodeId;
        // Optionally connect site to root domain node, if it exists
        if (groupHostnames.has(rootDomain)) {
          edges.push({ from: `domain:${rootDomain}`, to: siteNodeId });
        }
      }
    }

    const instanceNodeId = `instance:${server.ID || idx}:${server.url}`;
    const latestPing = server.ping_results && server.ping_results.length > 0
      ? server.ping_results.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())[0]
      : null;
    // Calculate isDown for backwards compatibility, but we'll use statusText to determine the actual state
    const isDown = !latestPing || latestPing.error || latestPing.status_code !== server.expected_status;
    const statusText = !latestPing ? 'Unknown' :
      (isDown ? 'Down' : 'Up');

    // Use statusText to determine up/down state instead of ping status
    const nodeIsDown = statusText.toLowerCase() === 'down';
    addNode(
      instanceNodeId,
      server.url,
      nodeIsDown ? 'down' : 'site', // Use statusText to determine the group
      `${server.url}\nStatus: ${statusText}\n${latestPing ? `Response: ${latestPing.status_code}` : ''}\n${server.comment ? `Note: ${server.comment}` : ''}`,
      nodeIsDown
    );
    edges.push({ from: parentForInstance, to: instanceNodeId });

    // --- Logging
    if (import.meta.env.MODE === 'development') {
      console.log(
        `Processed: ${hostname}\n` +
        `  → Parent: ${parentForInstance}\n` +
        `  → Domain nodes: [${[...groupHostnames].join(', ')}]`
      );
    }
    // ---

  } catch (error) {
    const errId = `error:${server.ID || idx}:${server.url}`;
    addNode(
      errId,
      server.url,
      'error',
      `${server.url}\nInvalid URL format\nMissing protocol (http:// or https://)?`
    );
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
