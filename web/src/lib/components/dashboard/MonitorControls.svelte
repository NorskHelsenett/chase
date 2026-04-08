<!-- MonitorControls.svelte -->
<script lang="ts">
	import ServerDialog from '../ServerDialog.svelte';
	import CustomSelect from '../ui/CustomSelect.svelte';
	import { Filter, Download, X, RefreshCw, Plus } from 'lucide-svelte';
	import { statusFilter } from '$lib/stores/filterStore';

	interface Props {
		isLoading?: boolean;
		onserverAdded?: () => void;
		onsearch?: (detail: { query: string }) => void;
		onrefresh?: () => void;
		onfilter?: (detail: { status: string }) => void;
		onexport?: () => void;
	}

	let { isLoading = false, onserverAdded, onsearch, onrefresh, onfilter, onexport }: Props = $props();

	let showDialog = $state(false);
	let searchQuery = $state('');

	// Handle dialog submission
	async function handleDialogSubmit(detail) {
		const { data, mode } = detail;

		try {
			const response = await fetch('/api/servers', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(data)
			});

			if (response.ok) {
				onserverAdded?.();
				showDialog = false;
				handleRefresh();
			} else {
				console.error('Failed to add server:', await response.text());
			}
		} catch (error) {
			console.error('Error adding server:', error);
		}
	}

	// Handle search
	function handleSearch() {
		onsearch?.({ query: searchQuery });
	}

	// Handle refresh
	function handleRefresh() {
		onrefresh?.();
	}

	// Handle filter change
	function handleFilterChange(event) {
		$statusFilter = event.value;
		onfilter?.({ status: event.value });
	}

	function onClose() {
		showDialog = false;
	}
</script>

<div class="controls-card">
	<div class="controls-row">
		<!-- Search -->
		<div class="search-wrapper">
			<input
				type="text"
				bind:value={searchQuery}
				oninput={handleSearch}
				placeholder="Search domains..."
				class="search-input"
			/>
			{#if searchQuery}
				<button
					class="search-clear"
					onclick={() => {
						searchQuery = '';
						handleSearch();
					}}
					aria-label="Clear search"
				>
					<X size={14} />
				</button>
			{/if}
		</div>

		<!-- Filter dropdown -->
		<div class="filter-wrapper">
			<CustomSelect
				bind:value={$statusFilter}
				icon={Filter}
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
				onchange={handleFilterChange}
			/>
		</div>

		<!-- Control buttons -->
		<div class="button-group">
			<button
				onclick={() => onexport?.()}
				disabled={isLoading}
				class="btn btn-secondary icon-only"
				title="Export current view as CSV"
			>
				<Download size={16} />
			</button>

			<button
				onclick={handleRefresh}
				disabled={isLoading}
				class="btn btn-secondary"
			>
				<RefreshCw size={16} class={isLoading ? 'spinning' : ''} />
				<span>{isLoading ? 'Refreshing...' : 'Refresh'}</span>
			</button>

			<button
				onclick={() => (showDialog = true)}
				disabled={isLoading}
				class="btn btn-primary"
			>
				<Plus size={16} />
				<span>Add Server</span>
			</button>
		</div>
	</div>
</div>

<ServerDialog
	bind:showDialog
	{isLoading}
	mode="add"
	initialData={null}
	onsubmit={handleDialogSubmit}
	onclose={onClose}
/>

<style>
	.controls-card {
		background: #202020;
		border-radius: 0.5rem;
		padding: 1rem;
		margin-bottom: 1rem;
	}

	.controls-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.search-wrapper {
		flex: 1;
		position: relative;
	}

	.search-input {
		width: 100%;
		padding: 0.5rem 1rem;
		background: #2b2b2b;
		border: none;
		border-radius: 0.5rem;
		color: #e5e7eb;
		font-size: 0.875rem;
		transition: box-shadow 0.15s ease;
	}

	.search-input::placeholder {
		color: #9ca3af;
	}

	.search-input:focus {
		outline: none;
		box-shadow: 0 0 0 2px #22c55e;
	}

	.search-clear {
		position: absolute;
		right: 0.5rem;
		top: 50%;
		transform: translateY(-50%);
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.5rem;
		height: 1.5rem;
		background: transparent;
		border: none;
		border-radius: 0.25rem;
		color: #9ca3af;
		cursor: pointer;
		transition: color 0.15s ease;
	}

	.search-clear:hover {
		color: #e5e7eb;
	}

	.filter-wrapper {
		position: relative;
		display: flex;
		align-items: center;
		z-index: 10;
	}

	.button-group {
		display: flex;
		gap: 0.75rem;
	}

	.btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		border: none;
		border-radius: 0.5rem;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s ease, opacity 0.15s ease;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-secondary {
		background: #2b2b2b;
		color: #e5e7eb;
	}

	.btn-secondary:hover:not(:disabled) {
		background: #333;
	}

	.btn-primary {
		background: #16a34a;
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background: #15803d;
	}

	.btn.icon-only {
		padding: 0.5rem;
	}

	:global(.spinning) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	@media (max-width: 768px) {
		.controls-row {
			flex-wrap: wrap;
		}

		.search-wrapper {
			width: 100%;
			flex: none;
		}

		.button-group {
			width: 100%;
			justify-content: flex-end;
		}
	}
</style>
