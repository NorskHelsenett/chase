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
		secretsRisk: 'critical' | 'high' | 'medium' | 'low' | '';
		secretsCount: number;
		days: DaySummary[];
		responseTimeMs: number | null;
	};

	interface Props {
		server: Server;
		hover?: boolean;
	}

	let { server, hover = false }: Props = $props();

	let rowData: ServerRowData = $derived(mapServerToRowData(server, pingInfo));

	let pulsing = $state(false);
	let lastTimestamp = $state('');

	$effect(() => {
		const ts = pingInfo?.latest?.timestamp;
		if (ts && lastTimestamp && ts !== lastTimestamp) {
			pulsing = true;
			setTimeout(() => { pulsing = false; }, 600);
		}
		if (ts) lastTimestamp = ts;
	});

	function mapServerToRowData(server: Server, pingInfo: any): ServerRowData {
		let status: 'up' | 'down' = 'down';
		let days: DaySummary[] = [];
		let responseTimeMs: number | null = null;

		if (pingInfo?.latest) {
			const s = pingInfo.latest.status_code;
			status = s > 0 && s === server.expected_status && !pingInfo.latest.error ? 'up' : 'down';
			responseTimeMs = pingInfo.latest.response_time_ms ?? null;
		} else {
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
			secretsRisk: server.secrets_risk?.toLowerCase() || '',
			secretsCount: server.secrets_count || 0,
			days,
			responseTimeMs
		};
	}

	function formatResponseTime(ms: number | null): { value: string; unit: string } | null {
		if (ms === null || ms === undefined) return null;
		if (ms >= 1000) return { value: (ms / 1000).toFixed(1), unit: 's' };
		return { value: String(Math.round(ms)), unit: 'ms' };
	}

	function getRiskClass(risk: string): string {
		switch (risk?.toLowerCase()) {
			case 'critical':
				return 'risk-critical';
			case 'high':
				return 'risk-high';
			case 'medium':
				return 'risk-medium';
			case 'low':
				return 'risk-low';
			default:
				return 'risk-none';
		}
	}

	function getScoreClass(score: string): string {
		switch (score) {
			case 'A+':
			case 'A':
				return 'score-a';
			case 'B+':
			case 'B':
				return 'score-b';
			case 'C':
				return 'score-c';
			case 'D':
			case 'F':
				return 'score-f';
			default:
				return 'score-none';
		}
	}

	function getDayBarClass(day: DaySummary): string {
		if (day.total === 0) return 'uptime-missing';
		if (day.uptime === 0) return 'uptime-down';
		if (day.uptime < 100) return 'uptime-missing';
		return 'uptime-up';
	}

	let pingInfo = $derived($pingData.get(server.ID));
</script>

<td class="cell cell-domain" class:hoverable={hover}>
	<div class="domain-info">
		<span class="status-dot-wrap">
			<span class="status-dot {rowData.status}" title={rowData.status.toUpperCase()}></span>
			{#if pulsing}
				<span class="pulse-ring {rowData.status}"></span>
			{/if}
		</span>
		{#if server.favicon}
			<img
				class="favicon"
				src={server.favicon.startsWith('http')
					? server.favicon
					: `https://${server.url}${server.favicon.startsWith('/') ? '' : '/'}${server.favicon}`}
				alt=""
				onerror={(e) => (e.currentTarget.style.display = 'none')}
			/>
		{/if}
		<div class="domain-text-wrap">
			<span class="domain-text">{rowData.title}</span>
			{#if server.site_title}
				<span class="site-title">{server.site_title}</span>
			{/if}
		</div>
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

<td class="cell" class:hoverable={hover}>
	{#if rowData.secretsCount > 0}
		<div class="risk-badge {getRiskClass(rowData.secretsRisk)}">
			{rowData.secretsCount}
		</div>
	{:else}
		<div class="risk-badge risk-none"></div>
	{/if}
</td>

<td class="cell cell-uptime" class:hoverable={hover}>
	<div class="uptime-bars">
		{#each Array(14) as _, i}
			{#if i < 14 - rowData.days.length}
				<div class="uptime-bar uptime-empty"></div>
			{:else}
				{@const day = rowData.days[i - (14 - rowData.days.length)]}
				<div class="uptime-bar-wrap">
					<div class="uptime-bar {getDayBarClass(day)}"></div>
					<div class="uptime-tooltip">
						<span class="tooltip-date">{day.date}</span>
						{#if day.total === 0}
							<span class="tooltip-detail">No pings</span>
						{:else}
							<span class="tooltip-uptime">{day.uptime.toFixed(1)}%</span>
							<span class="tooltip-detail">{day.successful}/{day.total} ok</span>
						{/if}
					</div>
				</div>
			{/if}
		{/each}
	</div>
</td>

<td class="cell cell-response-time" class:hoverable={hover}>
	{#if formatResponseTime(rowData.responseTimeMs)}
		{@const rt = formatResponseTime(rowData.responseTimeMs)}
		<span class="response-time"
			><span class="rt-value">{rt.value}</span><span class="rt-unit">{rt.unit}</span></span
		>
	{:else}
		<span class="response-time rt-unit">—</span>
	{/if}
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

	:global(tr:hover) .cell-domain.hoverable {
		border-radius: 0.5rem 0 0 0.5rem;
	}

	:global(tr:hover) .cell-response-time.hoverable {
		border-radius: 0 0.5rem 0.5rem 0;
	}

	.cell-domain {
		min-width: 300px;
		max-width: 300px;
		color: #e5e7eb;
	}

	.domain-info {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		min-width: 0;
	}

	.favicon {
		width: 22px;
		height: 22px;
		flex-shrink: 0;
		border-radius: 2px;
	}

	.domain-text-wrap {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.domain-text {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		font-size: 0.8125rem;
	}

	.site-title {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		font-size: 0.6875rem;
		color: #6b7280;
		line-height: 1.2;
	}

	.status-dot-wrap {
		position: relative;
		width: 10px;
		height: 10px;
		flex-shrink: 0;
		margin-right: 0.5rem;
	}

	.status-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		position: absolute;
		top: 0;
		left: 0;
	}

	.status-dot.up {
		background: #22c55e;
		box-shadow: 0 0 6px rgba(34, 197, 94, 0.6);
	}

	.status-dot.down {
		background: #ef4444;
		box-shadow: 0 0 6px rgba(239, 68, 68, 0.6);
	}

	.pulse-ring {
		position: absolute;
		top: 50%;
		left: 50%;
		width: 10px;
		height: 10px;
		border-radius: 50%;
		transform: translate(-50%, -50%);
		animation: pulse-fade 0.6s ease-out forwards;
		pointer-events: none;
	}

	.pulse-ring.up {
		border: 2px solid #22c55e;
	}

	.pulse-ring.down {
		border: 2px solid #ef4444;
	}

	@keyframes pulse-fade {
		0% {
			width: 10px;
			height: 10px;
			opacity: 0.7;
		}
		100% {
			width: 28px;
			height: 28px;
			opacity: 0;
		}
	}

	.cell-score {
		font-weight: 500;
	}

	.score-a {
		color: #22c55e;
	}
	.score-b {
		color: #eab308;
	}
	.score-c {
		color: #3b82f6;
	}
	.score-f {
		color: #ef4444;
	}
	.score-none {
		color: #6b7280;
	}

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

	.uptime-bar-wrap {
		position: relative;
	}

	.uptime-bar {
		width: 4px;
		height: 1rem;
		border-radius: 2px;
		transition: transform 0.15s ease, filter 0.15s ease;
	}

	.uptime-bar-wrap:hover .uptime-bar {
		transform: scaleY(1.4);
		filter: brightness(1.3);
	}

	.uptime-tooltip {
		display: none;
		position: absolute;
		bottom: calc(100% + 8px);
		left: 50%;
		transform: translateX(-50%);
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 6px;
		padding: 0.375rem 0.5rem;
		white-space: nowrap;
		z-index: 10;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		pointer-events: none;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
	}

	.uptime-tooltip::after {
		content: '';
		position: absolute;
		top: 100%;
		left: 50%;
		transform: translateX(-50%);
		border: 5px solid transparent;
		border-top-color: #333;
	}

	.uptime-bar-wrap:hover .uptime-tooltip {
		display: flex;
	}

	.tooltip-date {
		font-size: 0.625rem;
		color: #6b7280;
		letter-spacing: 0.02em;
	}

	.tooltip-uptime {
		font-size: 0.75rem;
		font-weight: 600;
		color: #e5e7eb;
	}

	.tooltip-detail {
		font-size: 0.625rem;
		color: #9ca3af;
	}

	.uptime-up {
		background: rgba(34, 197, 94, 0.7);
	}

	.uptime-missing {
		background: #eab308;
	}

	.uptime-down {
		background: #ef4444;
	}

	.uptime-empty {
		background: rgba(107, 114, 128, 0.15);
	}

	.cell-response-time {
		text-align: right;
		width: 1%;
		white-space: nowrap;
		padding-left: 0;
		padding-right: 0.25rem;
	}

	.response-time {
		font-size: 0.75rem;
		font-variant-numeric: tabular-nums;
	}

	.rt-value {
		color: #9ca3af;
	}

	.rt-unit {
		color: #4b5563;
	}
</style>
