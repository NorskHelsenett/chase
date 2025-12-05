<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import type { Server } from '$lib/models';
	import ScreenshotGrid from '$lib/components/grid/ScreenshotGrid.svelte';
	import CustomSelect from '$lib/components/ui/CustomSelect.svelte';
	import { Search, Grid, Filter } from 'lucide-svelte';
	import { servers, isLoading, serverStoreActions } from '$lib/stores/serverStore';

	let filteredServers: Server[] = [];
	let searchTerm = '';
	let filterStatus = 'online';
	let hasMounted = false;
	let lastActiveFilter: string | null | undefined = undefined;
	let activeFilter: string | null = null;

	function isSuccessfulStatus(status: number): boolean {
		return status >= 200 && status < 400;
	}

	function hasGoodPingHistory(server: Server): boolean {
		if (server.ping_results.length === 0) {
			return true; // New server with no pings
		}

		// Calculate success rate of all pings
		const successfulPings = server.ping_results.filter((ping) =>
			isSuccessfulStatus(ping.status_code)
		).length;

		const successRate = successfulPings / server.ping_results.length;
		return successRate >= 0.9; // 90% success rate threshold
	}

	$: activeFilter = $page.url.searchParams.get('active');

	$: {
		const allServers = $servers || [];
		if (allServers.length === 0) {
			filteredServers = [];
		} else {
			let result = allServers;
			if (activeFilter === 'true') {
				result = result.filter((server: Server) => server.active);
			} else if (activeFilter === 'false') {
				result = result.filter((server: Server) => !server.active);
			}

			if (searchTerm) {
				const term = searchTerm.toLowerCase();
				result = result.filter(
					(server: Server) =>
						server.url.toLowerCase().includes(term) ||
						server.name?.toLowerCase().includes(term) ||
						server.description?.toLowerCase().includes(term)
				);
			}

			if (filterStatus !== 'all') {
				if (filterStatus === 'online') {
					result = result.filter(
						(server: Server) => server.ping_results.length > 0 && hasGoodPingHistory(server)
					);
				} else if (filterStatus === 'issues') {
					result = result.filter(
						(server: Server) => server.ping_results.length > 0 && !hasGoodPingHistory(server)
					);
				} else if (filterStatus === 'new') {
					const thirtyDaysAgo = new Date();
					thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
					result = result.filter((server: Server) => new Date(server.CreatedAt) >= thirtyDaysAgo);
				}
			}

			filteredServers = result;
		}
	}

	async function loadData(force = false) {
		await serverStoreActions.setFilter(activeFilter ?? null, force);
	}

onMount(async () => {
	hasMounted = true;
});

$: if (hasMounted && activeFilter !== undefined && activeFilter !== lastActiveFilter) {
	lastActiveFilter = activeFilter;
	loadData();
}
</script>

<div class="p-4 min-h-screen w-full">
	<!-- Header and filters -->
	<div class="bg-[#202020] rounded-lg p-4 mb-4 flex flex-col gap-4">
		<div class="flex flex-wrap justify-between items-center gap-4">
			<h1 class="text-2xl font-medium flex items-center gap-2">
				<Grid size={24} class="text-green-500" />
				Server Grid View
				<span
					class="text-sm font-normal bg-green-500/20 text-green-300 px-3 py-1 rounded-full ml-2 shadow-inner"
				>
					{filteredServers.length}
					{filteredServers.length === 1 ? 'server' : 'servers'}
				</span>
			</h1>

			<div class="flex flex-wrap items-center gap-3">
				<div class="relative w-[400px]">
					<Search
						size={18}
						class="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-500 group-focus-within:text-green-500 transition-colors"
					/>
					<input
						type="text"
						bind:value={searchTerm}
						placeholder="Search servers..."
						class="w-full pl-10 pr-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-green-500 transition-all"
					/>
					{#if searchTerm}
						<button
							class="absolute right-2 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-200 transition-colors p-1"
							on:click={() => (searchTerm = '')}
							aria-label="Clear search"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								width="16"
								height="16"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<circle cx="12" cy="12" r="10" />
								<line x1="15" y1="9" x2="9" y2="15" />
								<line x1="9" y1="9" x2="15" y2="15" />
							</svg>
						</button>
					{/if}
				</div>

				<div class="relative flex items-center z-10">
					<CustomSelect
						bind:value={filterStatus}
						icon={Filter}
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
								value: 'issues',
								label: 'With issues',
								icon: '<div class="flex items-center"><span class="w-2 h-2 bg-red-400 rounded-full mr-2"></span><span class="text-red-400">Issues</span></div>'
							},
							{
								value: 'new',
								label: 'New',
								icon: '<div class="flex items-center"><span class="w-2 h-2 bg-gray-400 rounded-full mr-2"></span><span class="text-gray-300">New</span></div>'
							}
						]}
						on:change={() => {}}
					/>
				</div>
			</div>
		</div>
	</div>

	{#if $isLoading && filteredServers.length === 0}
		<div class="flex flex-col items-center justify-center py-16">
			<div class="relative">
				<div
					class="w-16 h-16 border-4 border-t-green-500 border-r-green-400/40 border-b-green-400/20 border-l-green-400/60 rounded-full animate-spin"
				></div>
				<div
					class="absolute inset-0 w-16 h-16 border-4 border-green-500/10 rounded-full animate-pulse"
				></div>
			</div>
			<p class="mt-5 text-green-400 font-medium">Loading servers...</p>
		</div>
	{:else}
		<ScreenshotGrid sites={filteredServers} />
	{/if}
</div>
