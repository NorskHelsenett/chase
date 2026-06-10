
<script lang="ts">
	import { run } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { page } from '$app/stores';
import CustomSelect from '$lib/components/ui/CustomSelect.svelte';
import { servers, serverStoreActions } from '$lib/stores/serverStore';
import { statusFilter } from '$lib/stores/filterStore';
import { pingData } from '$lib/stores/pingStore';
import { getEffectiveStatus } from '$lib/utils/status';
import { writable } from 'svelte/store';
import Graph from '$lib/components/dashboard/Graph.svelte';
import { Share2 } from 'lucide-svelte';
import type { Server, PingResult } from '$lib/models';
let hasMounted = $state(false);
let lastActiveFilter: string | null | undefined = $state(undefined);
let activeFilter: string | null = $state(null);

	const graphData = writable<{ nodes: GraphNode[]; edges: GraphEdge[] }>({ nodes: [], edges: [] });
	const isLoading = writable(true);

	interface GraphNode {
		id: string;
		label: string;
		group: string;
		title?: string;
		isDown?: boolean;
		cluster?: string;
		meta?: Record<string, unknown>;
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
		if (
			parts.length >= 3 &&
			parts[parts.length - 2].length <= 3 &&
			parts[parts.length - 1].length <= 3
		) {
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

// --- Graph util (core logic for buildGraphData) ---
function buildGraphData(serverList: Server[]) {
	const nodes: GraphNode[] = [];
	const edges: GraphEdge[] = [];

		const addedNodeIds = new Set();
		const groupHostnames = new Set();
		// Removed unused parentMap declaration

		// 1. Count all groupings
	const levelCounts = new Map<string, number>();
	serverList.forEach((server) => {
			try {
				const hostname = new URL(normalizeUrl(server.url)).hostname;
				getAllDomainLevels(hostname).forEach((level) => {
					levelCounts.set(level, (levelCounts.get(level) || 0) + 1);
				});
			} catch {}
		});

		function addNode(node: GraphNode) {
			if (!addedNodeIds.has(node.id)) {
				nodes.push({ ...node, cluster: node.cluster || node.label });
				addedNodeIds.add(node.id);
			}
		}

		// 2. Add domain/group nodes first and track hostnames
		for (const [level, count] of levelCounts) {
			if (count > 1) {
				addNode({
					id: `domain:${level}`,
					label: level,
					group: 'domain',
					title: `Domain/group: ${level}\nCount: ${count}`,
					cluster: getRootDomain(level),
					meta: { kind: 'domain', domain: level, count }
				});
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

		// 4. Add server nodes, connecting to their closest group
	serverList.forEach((server, idx) => {
			try {
				const serverUrl = normalizeUrl(server.url);
				const url = new URL(serverUrl);
				const hostname = url.hostname;
				const allLevels = getAllDomainLevels(hostname);
				const rootDomain = getRootDomain(hostname);

				const effectiveStatus = getEffectiveStatus(server);
				const isDown = effectiveStatus === 'down';
				const statusText = effectiveStatus === 'up' ? 'Up' : 'Down';
				const nodeId = `instance:${server.ID || idx}:${server.url}`;
				const tooltip = `${server.url}\nStatus: ${statusText}${server.comment ? `\nNote: ${server.comment}` : ''}`;
				const latest = server.ping_results?.[0];

				addNode({
					id: nodeId,
					label: server.url,
					group: isDown ? 'error' : 'site',
					title: tooltip,
					isDown,
					cluster: rootDomain,
					meta: {
						kind: 'site',
						url: server.url,
						status: effectiveStatus,
						statusCode: latest?.status_code,
						responseTimeMs: latest?.response_time_ms,
						error: latest?.error,
						note: server.comment
					}
				});

				// Find the closest parent domain/group node
				let parent = null;
				for (let i = 0; i < allLevels.length; i++) {
					if (groupHostnames.has(allLevels[i])) {
						parent = `domain:${allLevels[i]}`;
						break;
					}
				}
				if (!parent && groupHostnames.has(rootDomain)) {
					parent = `domain:${rootDomain}`;
				}

				if (parent) {
					edges.push({ from: parent, to: nodeId });
				}
			} catch (error) {
				addNode({
					id: `error:${server.ID || idx}:${server.url}`,
					label: server.url,
					group: 'error',
					title: `${server.url}\nInvalid URL format`,
					isDown: true,
					cluster: server.url,
					meta: { kind: 'invalid', url: server.url }
				});
			}
		});

		return { nodes, edges };
	}

	run(() => {
		activeFilter = $page.url.searchParams.get('active') ?? 'true';
	});

	onMount(() => {
		hasMounted = true;
	});

async function loadGraphData(force = false) {
	isLoading.set(true);
	try {
		await serverStoreActions.setFilter(activeFilter ?? null, force);
		updateGraph();
	} finally {
		isLoading.set(false);
	}
}

	run(() => {
		if (hasMounted && activeFilter !== lastActiveFilter) {
			lastActiveFilter = activeFilter;
			loadGraphData();
		}
	});

function updateGraph() {
	let serverList: Server[] = ($servers || []).map((server) => {
		const info = $pingData.get(server.ID);
		if (info?.latest) {
			// Inject latest ping as a single-element array for graph compatibility
			return { ...server, ping_results: [{
				status_code: info.latest.status_code,
				response_time_ms: info.latest.response_time_ms,
				error: info.latest.error || '',
				timestamp: info.latest.timestamp
			}] };
		}
		return { ...server, ping_results: server.ping_results || [] };
	});
		if ($statusFilter !== 'all') {
			if ($statusFilter === 'online') {
				serverList = serverList.filter((server: Server) => getEffectiveStatus(server) === 'up');
			} else if ($statusFilter === 'issues' || $statusFilter === 'offline') {
				serverList = serverList.filter((server: Server) => getEffectiveStatus(server) === 'down');
			} else if ($statusFilter === 'new') {
				const thirtyDaysAgo = new Date();
				thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
				serverList = serverList.filter(
					(server: Server) => new Date(server.CreatedAt) >= thirtyDaysAgo
				);
			}
		}
		graphData.set(buildGraphData(serverList));
	}
</script>

<div class="p-4 w-full h-full">
	<div class="bg-[#202020] rounded-lg p-4 mb-4 gap-4 flex flex-wrap justify-between items-center gap-4 mb-4">
		<h1 class="text-2xl font-medium flex items-center gap-2">
			<Share2 size={24} class="text-green-500" />
			Site Graph View
		</h1>
		<div class="flex flex-wrap items-center gap-3">
			<div class="relative flex items-center z-10">
				<CustomSelect
					bind:value={$statusFilter}
					storageKey="chase-filter-status"
					options={[
						{
							value: 'all',
							label: 'All servers',
							icon: '<div class="flex items-center"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg><span class="text-gray-100 ml-2"> Show all</span></div>'
						},
						{
							value: 'online',
							label: 'Online',
							icon: '<div class="flex items-center"><span class="w-2 h-2 bg-green-400 rounded-full mr-2 animate-pulse"></span><span class="text-green-400">Online</span></div>'
						},
						{
							value: 'offline',
							label: 'Offline',
							icon: '<div class="flex items-center"><span class="w-2 h-2 bg-red-400 rounded-full mr-2"></span><span class="text-red-400">Offline</span></div>'
						},
						{
							value: 'new',
							label: 'New',
							icon: '<div class="flex items-center"><span class="w-2 h-2 bg-gray-400 rounded-full mr-2"></span><span class="text-gray-300">New</span></div>'
						}
					]}
					onchange={() => { updateGraph(); }}
				/>
			</div>
		</div>
	</div>
	{#if $isLoading}
		<div class="flex justify-center items-center p-6">
			<div class="animate-pulse text-gray-500">Loading graph data...</div>
		</div>
	{:else}
		<Graph {graphData} />
	{/if}
</div>
