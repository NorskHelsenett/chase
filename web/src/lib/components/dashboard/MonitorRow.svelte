<script lang="ts">
	import type { Server } from '$lib/models';
	import { pingData } from '$lib/stores/pingStore';

	type DaySummary = { date: string; total: number; successful: number; uptime: number };

	type ServerRowData = {
		status: 'up' | 'down';
		title: string;
		headerScore: 'A+' | 'A' | 'B' | 'C' | 'D' | 'F' | '';
		certScore: 'A+' | 'A' | 'B' | 'C' | 'D' | 'F' | '';
		adminRisk: 'critical' | 'high' | 'medium' | 'low' | '';
		apiRisk: 'critical' | 'high' | 'medium' | 'low' | '';
		days: DaySummary[];
	};

	export let server: Server;
	export let hover = false;

	let rowData: ServerRowData;

	$: pingInfo = $pingData.get(server.ID);

	$: rowData = mapServerToRowData(server, pingInfo);

	function mapServerToRowData(server: Server, pingInfo: any): ServerRowData {
		let status: 'up' | 'down' = 'down';
		let days: DaySummary[] = [];

		if (pingInfo?.latest) {
			// SSE has real-time data — use it for live status
			const s = pingInfo.latest.status_code;
			status = s > 0 && s === server.expected_status && !pingInfo.latest.error ? 'up' : 'down';
		} else {
			// Use API-provided status (available immediately)
			status = server.status === 'up' ? 'up' : 'down';
		}

		if (pingInfo?.days) {
			days = pingInfo.days;
		}

		return {
			status,
			title: server.url,
			headerScore: server.header_score || server.security?.headerRisk || '',
			certScore: server.cert_score || server.security?.certRisk || '',
			adminRisk: server.admin_risk?.toLowerCase() || server.security?.adminRisk || '',
			apiRisk: server.api_risk?.toLowerCase() || server.security?.apiRisk || '',
			days
		};
	}

	function getRiskClass(risk: string): string {
		switch (risk?.toLowerCase()) {
			case 'critical': return 'risk-critical';
			case 'high': return 'risk-high';
			case 'medium': return 'risk-medium';
			case 'low': return 'risk-low';
			default: return 'risk-none';
		}
	}

	function getScoreClass(score: string): string {
		switch (score) {
			case 'A+':
			case 'A': return 'score-a';
			case 'B+':
			case 'B': return 'score-b';
			case 'C': return 'score-c';
			case 'D':
			case 'F': return 'score-f';
			default: return 'score-none';
		}
	}

	function getDayBarClass(day: DaySummary): string {
		if (day.uptime >= 99.9) return 'uptime-up';
		if (day.uptime >= 95) return 'uptime-degraded';
		return 'uptime-down';
	}

	function formatDayTooltip(day: DaySummary): string {
		return `${day.date}\n${day.uptime.toFixed(1)}% (${day.successful}/${day.total})`;
	}
</script>

<td class="cell cell-status" class:hoverable={hover}>
	<span class="status-badge {rowData.status}">
		{rowData.status.toUpperCase()}
	</span>
</td>

<td class="cell cell-domain" class:hoverable={hover}>
	<div class="domain-text">
		{rowData.title}
	</div>
</td>

<td class="cell cell-score {getScoreClass(rowData.headerScore)}" class:hoverable={hover}>
	{rowData.headerScore}
</td>

<td class="cell cell-score {getScoreClass(rowData.certScore)}" class:hoverable={hover}>
	{rowData.certScore}
</td>

<td class="cell" class:hoverable={hover}>
	<div class="risk-badge {getRiskClass(rowData.adminRisk)}">
		{rowData.adminRisk || ''}
	</div>
</td>

<td class="cell" class:hoverable={hover}>
	<div class="risk-badge {getRiskClass(rowData.apiRisk)}">
		{rowData.apiRisk || ''}
	</div>
</td>

<td class="cell cell-uptime" class:hoverable={hover}>
	<div class="uptime-bars">
		{#each Array(10) as _, i}
			{#if i < 10 - rowData.days.length}
				<div class="uptime-bar uptime-empty"></div>
			{:else}
				{@const day = rowData.days[i - (10 - rowData.days.length)]}
				<div
					class="uptime-bar {getDayBarClass(day)}"
					title={formatDayTooltip(day)}
				></div>
			{/if}
		{/each}
	</div>
</td>

<style>
	.cell {
		padding: 0.5rem;
		transition: background-color 0.15s ease;
	}

	.cell.hoverable {
		/* Hover handled by parent tr via :global */
	}

	:global(tr:hover) .cell.hoverable {
		background: #2b2b2b;
	}

	:global(tr:hover) .cell-status.hoverable {
		border-radius: 0.5rem 0 0 0.5rem;
	}

	:global(tr:hover) .cell-uptime.hoverable {
		border-radius: 0 0.5rem 0.5rem 0;
	}

	.cell-domain {
		min-width: 300px;
		max-width: 300px;
		color: #e5e7eb;
	}

	.domain-text {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		width: 100%;
		max-width: 100%;
	}

	.status-badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0.125rem 0.5rem;
		min-width: 5em;
		font-size: 0.6875rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.025em;
		border-radius: 9999px;
		border: 1px solid;
	}

	.status-badge.up {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
		border-color: rgba(34, 197, 94, 0.3);
	}

	.status-badge.down {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
		border-color: rgba(239, 68, 68, 0.3);
	}

	.cell-score {
		font-weight: 500;
	}

	.score-a { color: #22c55e; }
	.score-b { color: #eab308; }
	.score-c { color: #3b82f6; }
	.score-f { color: #ef4444; }
	.score-none { color: #6b7280; }

	.risk-badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0.25rem 0.5rem;
		min-width: 5em;
		font-size: 0.75rem;
		font-weight: 500;
		text-transform: capitalize;
		border-radius: 9999px;
	}

	.risk-critical {
		background: rgba(239, 68, 68, 0.15);
		color: #ef4444;
	}

	.risk-high {
		background: rgba(249, 115, 22, 0.15);
		color: #f97316;
	}

	.risk-medium {
		background: rgba(234, 179, 8, 0.15);
		color: #eab308;
	}

	.risk-low {
		background: rgba(34, 197, 94, 0.15);
		color: #22c55e;
	}

	.risk-none {
		background: rgba(107, 114, 128, 0.15);
		color: #6b7280;
	}

	.uptime-bars {
		display: flex;
		gap: 2px;
	}

	.uptime-bar {
		width: 4px;
		height: 1rem;
		border-radius: 2px;
	}

	.uptime-up {
		background: rgba(163, 230, 53, 0.7);
	}

	.uptime-degraded {
		background: #eab308;
	}

	.uptime-down {
		background: #ef4444;
	}

	.uptime-empty {
		background: rgba(34, 197, 94, 0.1);
	}
</style>
