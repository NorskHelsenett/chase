<script lang="ts">
	import MonitorRow from './MonitorRow.svelte';
	import type { Server } from '$lib/models';
	import { goto } from '$app/navigation';
	import { fade, scale } from 'svelte/transition';
	import { Monitor } from 'lucide-svelte';

	export let sites: Server[] = [];

	let sortField:
		| keyof Server
		| 'status'
		| 'header'
		| 'cert'
		| 'adminRisk'
		| 'apiRisk'
		| 'uptime'
		| null = null;
	let sortDirection: 'asc' | 'desc' = 'asc';

	function gradeToNumber(grade: string): number {
		const grades = {
			'A+': 7,
			A: 6,
			'B+': 5,
			B: 4,
			C: 3,
			D: 2,
			F: 1,
			'': 0
		};
		return grades[grade as keyof typeof grades] || 0;
	}

	function riskToNumber(risk: string): number {
		const risks = {
			critical: 4,
			high: 3,
			medium: 2,
			low: 1,
			'': 0
		};
		return risks[risk.toLowerCase() as keyof typeof risks] || 0;
	}

	function getUptimePercentage(server: Server): number {
		const pings = server.ping_results || [];
		if (pings.length === 0) return 0;

		const successfulPings = pings.filter(
			(ping) => ping.status_code === server.expected_status
		).length;

		return (successfulPings / pings.length) * 100;
	}

	function toggleSort(field: typeof sortField) {
		if (sortField === field) {
			sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
		} else {
			sortField = field;
			sortDirection = 'asc';
		}

		sites = [...sites].sort((a, b) => {
			let valueA, valueB;

			switch (field) {
				case 'status':
					valueA = getLatestPingStatus(a);
					valueB = getLatestPingStatus(b);
					break;
				case 'url':
					valueA = a.url.toLowerCase();
					valueB = b.url.toLowerCase();
					break;
				case 'header':
				case 'cert':
					valueA = gradeToNumber(getLatestGrade(a, field));
					valueB = gradeToNumber(getLatestGrade(b, field));
					break;
				case 'adminRisk':
				case 'apiRisk':
					valueA = riskToNumber(getLatestRisk(a, field));
					valueB = riskToNumber(getLatestRisk(b, field));
					break;
				case 'ip':
					valueA = a.ping_results?.[0]?.ip || '';
					valueB = b.ping_results?.[0]?.ip || '';
					break;
				case 'uptime':
					valueA = getUptimePercentage(a);
					valueB = getUptimePercentage(b);
					break;
				default:
					valueA = a[field as keyof Server];
					valueB = b[field as keyof Server];
			}

			if (valueA < valueB) return sortDirection === 'asc' ? -1 : 1;
			if (valueA > valueB) return sortDirection === 'asc' ? 1 : -1;
			return 0;
		});
	}

	function getLatestPingStatus(server: Server): boolean {
		const sortedPings = [...(server.ping_results || [])].sort(
			(a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
		);
		return !sortedPings.length || sortedPings[0]?.error || sortedPings[0]?.status_code >= 400
			? false
			: true;
	}

	function getLatestGrade(server: Server, type: 'header' | 'cert'): string {
		if (type === 'header') {
			return server.header_score || server.security?.headerRisk || '';
		} else {
			return server.cert_score || server.security?.certRisk || '';
		}
	}

	function getLatestRisk(server: Server, type: 'adminrisk' | 'apirisk'): string {
		if (type === 'adminrisk') {
			return server.admin_risk?.toLowerCase() || server.security?.adminRisk || '';
		} else {
			return server.api_risk?.toLowerCase() || server.security?.apiRisk || '';
		}
	}

	function getSortIndicator(field: typeof sortField): string {
		if (sortField !== field) return '↕';
		return sortDirection === 'asc' ? '↑' : '↓';
	}
</script>

<div class="table-card">
	{#if sites.length === 0}
		<div class="empty-state">
			<div class="empty-icon" in:scale={{ duration: 400 }}>
				<Monitor size={32} />
			</div>
			<p class="empty-title" in:fade={{ duration: 300, delay: 100 }}>
				No servers found
			</p>
			<p class="empty-description" in:fade={{ duration: 300, delay: 200 }}>
				Try adjusting your search filters or add a new server to monitor
			</p>
		</div>
	{:else}
		<table class="monitor-table">
			<thead>
				<tr>
					<th class="sortable" on:click={() => toggleSort('status')}>
						<span>Status</span>
						<span class="sort-indicator">{getSortIndicator('status')}</span>
					</th>
					<th class="sortable col-domain" on:click={() => toggleSort('url')}>
						<span>Domain</span>
						<span class="sort-indicator">{getSortIndicator('url')}</span>
					</th>
					<th class="sortable" on:click={() => toggleSort('header')}>
						<span>Header</span>
						<span class="sort-indicator">{getSortIndicator('header')}</span>
					</th>
					<th class="sortable" on:click={() => toggleSort('cert')}>
						<span>Cert</span>
						<span class="sort-indicator">{getSortIndicator('cert')}</span>
					</th>
					<th class="sortable" on:click={() => toggleSort('adminRisk')}>
						<span>Admin Risk</span>
						<span class="sort-indicator">{getSortIndicator('adminRisk')}</span>
					</th>
					<th class="sortable" on:click={() => toggleSort('apiRisk')}>
						<span>API Risk</span>
						<span class="sort-indicator">{getSortIndicator('apiRisk')}</span>
					</th>
					<th class="sortable" on:click={() => toggleSort('uptime')}>
						<span>Uptime</span>
						<span class="sort-indicator">{getSortIndicator('uptime')}</span>
					</th>
				</tr>
			</thead>
			<tbody>
				{#each sites as site}
					<tr
						data-server-id={site.ID}
						class="clickable-row"
						on:click={() => goto(`/server/${site.ID}`)}
					>
						<MonitorRow server={site} hover={true} />
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>

<style>
	.table-card {
		background: #202020;
		border-radius: 0.5rem;
		padding: 1rem;
	}

	.empty-state {
		padding: 4rem 1rem;
		text-align: center;
	}

	.empty-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 4rem;
		height: 4rem;
		margin: 0 auto 1.25rem;
		border-radius: 50%;
		background: rgba(34, 197, 94, 0.1);
		color: rgba(34, 197, 94, 0.7);
	}

	.empty-title {
		font-size: 1.125rem;
		font-weight: 500;
		color: #4ade80;
		margin-bottom: 0.5rem;
	}

	.empty-description {
		font-size: 0.875rem;
		color: #9ca3af;
		max-width: 28rem;
		margin: 0 auto;
	}

	.monitor-table {
		width: 100%;
		border-collapse: separate;
		border-spacing: 0 0.25rem;
	}

	.monitor-table thead tr {
		color: #9ca3af;
	}

	.monitor-table th {
		text-align: left;
		padding: 0.5rem;
		font-size: 0.75rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.025em;
	}

	.monitor-table th.sortable {
		cursor: pointer;
		transition: color 0.15s ease;
		user-select: none;
	}

	.monitor-table th.sortable:hover {
		color: #e5e7eb;
	}

	.monitor-table th.col-domain {
		width: 30%;
	}

	.sort-indicator {
		margin-left: 0.25rem;
		opacity: 0;
		transition: opacity 0.15s ease;
	}

	.monitor-table th.sortable:hover .sort-indicator {
		opacity: 1;
	}

	.clickable-row {
		cursor: pointer;
		transition: background-color 0.15s ease;
	}
</style>
