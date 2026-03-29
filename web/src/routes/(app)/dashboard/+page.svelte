<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import MonitorStats from '$lib/components/dashboard/MonitorStats.svelte';
	import MonitorControls from '$lib/components/dashboard/MonitorControls.svelte';
	import MonitorTable from '$lib/components/dashboard/MonitorTable.svelte';
	import { servers, isLoading, serverStoreActions } from '$lib/stores/serverStore';
	import { pingData } from '$lib/stores/pingStore';
	import type { Server } from '$lib/models';
	import { exportServersToCSV } from '$lib/utils/csv.js';

	let filteredServers: Server[] = [];

	function hasGoodPingHistory(server: Server): boolean {
		if (!server.ping_results || server.ping_results.length === 0) {
			return true; // New server with no pings
		}

		const successfulPings = server.ping_results.filter((ping) =>
			ping.status_code === server.expected_status
		).length;

		const successRate = successfulPings / server.ping_results.length;
		return successRate >= 0.9; // 90% success rate threshold
	}
	let searchQuery = '';
	let statusFilter = 'all';
	let hasMounted = false;
	let lastActiveFilter: string | null | undefined = undefined;
	let activeFilter: string | null = null;

	// Subscribe to page store to get URL parameters
	$: activeFilter = $page.url.searchParams.get('active');

	// Merge ping data from SSE into servers
	$: serversWithPings = $servers.map((server) => {
		const pings = $pingData.get(server.ID);
		if (pings && pings.length > 0) {
			return { ...server, ping_results: pings };
		}
		return server;
	});

	// Compute stats from servers with ping data
	$: stats = serversWithPings.reduce(
		(acc, server) => {
			const sortedPings = [...(server.ping_results || [])].sort(
				(a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
			);
			const latestPing = sortedPings[0];

			if (latestPing) {
				if (latestPing.status_code === server.expected_status) {
					acc.up += 1;
				} else {
					acc.down += 1;
				}
				if (server.security_risk_level === 'CRITICAL') {
					acc.criticalRisks += 1;
				} else if (server.security_risk_level === 'HIGH') {
					acc.highRisks += 1;
				}
			} else {
				acc.down += 1;
			}

			return acc;
		},
		{ up: 0, down: 0, criticalRisks: 0, highRisks: 0 }
	);

	// Filter servers based on search query and status
	$: {
		let result = serversWithPings;

		if (searchQuery) {
			const query = searchQuery.toLowerCase();
			result = result.filter(
				(server) =>
					server.url.toLowerCase().includes(query) || server.comment?.toLowerCase().includes(query)
			);
		}

		if (statusFilter !== 'all') {
			if (statusFilter === 'online') {
				result = result.filter((server) => hasGoodPingHistory(server));
			} else if (statusFilter === 'issues') {
				result = result.filter((server) => !hasGoodPingHistory(server));
			} else if (statusFilter === 'new') {
				const thirtyDaysAgo = new Date();
				thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
				result = result.filter((server) => new Date(server.CreatedAt) >= thirtyDaysAgo);
			}
		}

		filteredServers = result;
	}

async function fetchServers(forceRefresh = false) {
	await serverStoreActions.setFilter(activeFilter ?? null, forceRefresh);
}

	function handleSearch(event: CustomEvent) {
		searchQuery = event.detail.query.toLowerCase();
	}

	function handleRefresh() {
		fetchServers(true); // Force refresh from server
	}

	function handleFilter(event: CustomEvent) {
		statusFilter = event.detail.status;
	}

	// Handle CSV export
	function handleExport() {
		// Create a filename based on current filters and date
		const date = new Date().toISOString().split('T')[0];
		const filterName = statusFilter !== 'all' ? `-${statusFilter}` : '';
		const searchSuffix = searchQuery ? `-search_${searchQuery}` : '';
		const filename = `server-data${filterName}${searchSuffix}-${date}.csv`;

		// Export current filtered view to CSV
		exportServersToCSV(filteredServers, filename);
	}

onMount(() => {
	hasMounted = true;
});

$: if (hasMounted && activeFilter !== undefined && activeFilter !== lastActiveFilter) {
	lastActiveFilter = activeFilter;
	fetchServers();
}
</script>

<div class="p-4 w-full">
	<MonitorStats {stats} />
	<MonitorControls
		isLoading={$isLoading}
		on:search={handleSearch}
		on:refresh={handleRefresh}
		on:filter={handleFilter}
		on:export={handleExport}
		on:serverAdded={() => fetchServers(true)}
	/>
	{#if $isLoading && filteredServers.length === 0}
		<div class="flex justify-center items-center p-6">
			<div class="animate-pulse text-gray-500">Loading server data...</div>
		</div>
	{:else}
		<MonitorTable sites={filteredServers} />
	{/if}
</div>
