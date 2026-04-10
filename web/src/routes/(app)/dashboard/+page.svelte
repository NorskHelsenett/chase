<script lang="ts">
	import { run } from 'svelte/legacy';

	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import MonitorStats from '$lib/components/dashboard/MonitorStats.svelte';
	import MonitorControls from '$lib/components/dashboard/MonitorControls.svelte';
	import MonitorTable from '$lib/components/dashboard/MonitorTable.svelte';
	import { servers, isLoading, serverStoreActions } from '$lib/stores/serverStore';
	import { statusFilter } from '$lib/stores/filterStore';
	import { pingData } from '$lib/stores/pingStore';
	import type { Server } from '$lib/models';
	import { exportServersToCSV } from '$lib/utils/csv.js';
	import { getEffectiveStatus } from '$lib/utils/status';

	let filteredServers: Server[] = $state([]);

	let searchQuery = $state('');
	let hasMounted = $state(false);
	let lastActiveFilter: string | null | undefined = $state(undefined);
	let activeFilter: string | null = $state(null);

	run(() => {
		activeFilter = $page.url.searchParams.get('active');
	});

	// Filter servers based on search query and status
	run(() => {
		// Reference pingData so this block re-runs when SSE updates arrive
		void $pingData;
		let result = $servers;

		if (searchQuery) {
			const query = searchQuery.toLowerCase();
			result = result.filter(
				(server) =>
					server.url.toLowerCase().includes(query) ||
					server.comment?.toLowerCase().includes(query) ||
					server.site_title?.toLowerCase().includes(query) ||
					server.site_description?.toLowerCase().includes(query)
			);
		}

		if ($statusFilter !== 'all') {
			if ($statusFilter === 'online') {
				result = result.filter((server) => getEffectiveStatus(server) === 'up');
			} else if ($statusFilter === 'issues') {
				result = result.filter((server) => getEffectiveStatus(server) === 'down');
			} else if ($statusFilter === 'new') {
				const thirtyDaysAgo = new Date();
				thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
				result = result.filter((server) => new Date(server.CreatedAt) >= thirtyDaysAgo);
			}
		}

		filteredServers = result;
	});

	// Compute stats from the filtered list using SSE-aware status
	let stats = $derived(filteredServers.reduce(
		(acc, server) => {
			if (getEffectiveStatus(server) === 'up') {
				acc.up += 1;
			} else {
				acc.down += 1;
			}
			if (server.secrets_count && server.secrets_count > 0) {
				acc.secretsExposed += server.secrets_count;
			}
			if (server.security_risk_level === 'CRITICAL' || server.security_risk_level === 'HIGH') {
				acc.highRisks += 1;
			}
			return acc;
		},
		{ up: 0, down: 0, secretsExposed: 0, highRisks: 0 }
	));

	async function fetchServers(forceRefresh = false) {
		await serverStoreActions.setFilter(activeFilter ?? null, forceRefresh);
	}

	function handleSearch(detail) {
		searchQuery = detail.query.toLowerCase();
	}

	function handleRefresh() {
		fetchServers(true);
	}

	function handleFilter(detail) {
		$statusFilter = detail.status;
	}

	function handleExport() {
		const date = new Date().toISOString().split('T')[0];
		const filterName = $statusFilter !== 'all' ? `-${$statusFilter}` : '';
		const searchSuffix = searchQuery ? `-search_${searchQuery}` : '';
		const filename = `server-data${filterName}${searchSuffix}-${date}.csv`;
		exportServersToCSV(filteredServers, filename);
	}

	onMount(() => {
		hasMounted = true;
	});

	run(() => {
		if (hasMounted && activeFilter !== undefined && activeFilter !== lastActiveFilter) {
			lastActiveFilter = activeFilter;
			fetchServers();
		}
	});
</script>

<div class="dashboard">
	<div class="dashboard-header">
		<MonitorStats {stats} />
		<MonitorControls
			isLoading={$isLoading}
			onsearch={handleSearch}
			onrefresh={handleRefresh}
			onfilter={handleFilter}
			onexport={handleExport}
			onserverAdded={() => fetchServers(true)}
		/>
	</div>
	<div class="dashboard-table">
		{#if $isLoading && filteredServers.length === 0}
			<div class="flex justify-center items-center p-6">
				<div class="animate-pulse text-gray-500">Loading server data...</div>
			</div>
		{:else}
			<MonitorTable sites={filteredServers} />
		{/if}
	</div>
</div>

<style>
	.dashboard {
		display: flex;
		flex-direction: column;
		height: 100%;
		padding: 1rem;
		box-sizing: border-box;
		overflow: hidden;
	}

	.dashboard-header {
		flex-shrink: 0;
	}

	.dashboard-table {
		flex: 1;
		min-height: 0;
		border-radius: 0.5rem;
		overflow: auto;
		scrollbar-width: thin;
		scrollbar-color: #333 transparent;
	}

	.dashboard-table::-webkit-scrollbar {
		width: 6px;
	}

	.dashboard-table::-webkit-scrollbar-track {
		background: transparent;
	}

	.dashboard-table::-webkit-scrollbar-thumb {
		background: #333;
		border-radius: 3px;
	}
</style>
