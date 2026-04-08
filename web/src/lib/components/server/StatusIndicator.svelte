<script lang="ts">
	import { run } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { computePosition, autoPlacement, offset, shift, arrow } from '@floating-ui/dom';
	import type { PingResult } from '$lib/models';

	interface Props {
		pingResults?: PingResult[];
		expectedStatus?: number;
	}

	let { pingResults = [], expectedStatus = 200 }: Props = $props();

	const MAX_DAYS = 90;

	let status: 'up' | 'down' = $state('down');
	let tooltipElement: HTMLDivElement = $state();
	let arrowElement: HTMLDivElement = $state();
	let currentTarget: HTMLElement | null = null;

	type AggregatedDay = {
		date: string;
		uptime: number;
		totalPings: number;
		successfulPings: number;
	};

	let aggregatedDays: AggregatedDay[] = $state([]);

	const pingSuccessful = (ping: PingResult) =>
		ping.status_code > 0 && ping.status_code === expectedStatus;

	run(() => {
		// Always aggregate by day
		const dailyMap: Record<string, { total: number; success: number }> = {};

		for (const ping of pingResults) {
			const date = new Date(ping.timestamp).toISOString().split('T')[0];
			if (!dailyMap[date]) {
				dailyMap[date] = { total: 0, success: 0 };
			}
			dailyMap[date].total++;
			if (pingSuccessful(ping)) {
				dailyMap[date].success++;
			}
		}

		// Build a continuous range of days (last MAX_DAYS)
		const today = new Date();
		const allDays: AggregatedDay[] = [];
		for (let i = MAX_DAYS - 1; i >= 0; i--) {
			const d = new Date(today);
			d.setDate(d.getDate() - i);
			const date = d.toISOString().split('T')[0];
			const entry = dailyMap[date];
			allDays.push({
				date,
				totalPings: entry?.total ?? 0,
				successfulPings: entry?.success ?? 0,
				uptime: entry && entry.total > 0 ? (entry.success / entry.total) * 100 : -1
			});
		}
		aggregatedDays = allDays;
	});

	// Status based on the most recent day
	run(() => {
		const latest = aggregatedDays.length > 0 ? aggregatedDays[aggregatedDays.length - 1] : null;
		status = latest && latest.uptime >= 99.9 ? 'up' : 'down';
	});

	const getStatusColor = (uptime: number) => {
		if (uptime < 0) return 'empty';
		if (uptime >= 99.9) return 'bar-perfect';
		if (uptime >= 99.0) return 'bar-good';
		if (uptime >= 95) return 'bar-degraded';
		return 'bar-down';
	};

	const formatTooltipContent = (day: AggregatedDay) => {
		const d = new Date(day.date + 'T00:00:00');
		const label = d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
		return `${label} — ${day.uptime.toFixed(1)}%\n${day.successfulPings}/${day.totalPings} checks passed`;
	};

	const showTooltip = async (event: MouseEvent, day: AggregatedDay) => {
		const target = event.currentTarget as HTMLElement;
		currentTarget = target;
		tooltipElement.style.display = 'block';
		tooltipElement.textContent = formatTooltipContent(day);

		const { x, y, placement, middlewareData } = await computePosition(target, tooltipElement, {
			placement: 'top',
			middleware: [
				offset(8),
				autoPlacement({ allowedPlacements: ['top', 'bottom'] }),
				shift({ padding: 5 }),
				arrow({ element: arrowElement })
			]
		});

		Object.assign(tooltipElement.style, { left: `${x}px`, top: `${y}px` });

		const { x: arrowX, y: arrowY } = middlewareData.arrow;
		const staticSide = { top: 'bottom', bottom: 'top' }[placement.split('-')[0]];
		Object.assign(arrowElement.style, {
			left: arrowX != null ? `${arrowX}px` : '',
			top: arrowY != null ? `${arrowY}px` : '',
			right: '',
			bottom: '',
			[staticSide]: '-4px'
		});
	};

	const hideTooltip = () => {
		if (tooltipElement) tooltipElement.style.display = 'none';
		currentTarget = null;
	};

	onMount(() => () => hideTooltip());
</script>

<div class="status-container">
	<div class="status-bars">
		{#each aggregatedDays as day}
			<div
				class="status-bar {getStatusColor(day.uptime)}"
				onmouseenter={(e) => day.totalPings > 0 && showTooltip(e, day)}
				onmouseleave={hideTooltip}
			></div>
		{/each}
	</div>

	<div class="status-indicator {status}">
		<span class="status-dot"></span>
		<span class="status-text">{status === 'up' ? 'Operational' : 'Degraded'}</span>
	</div>
</div>

<div class="tooltip" bind:this={tooltipElement}>
	<div class="tooltip-arrow" bind:this={arrowElement}></div>
</div>

<style>
	.status-container {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.5rem 0;
	}

	.status-bars {
		display: flex;
		gap: 2px;
		flex: 1;
	}

	.status-bar {
		flex: 1;
		min-width: 0;
		height: 24px;
		border-radius: 2px;
		transition: opacity 0.15s ease;
		cursor: pointer;
	}

	.status-bar:hover {
		opacity: 0.75;
	}

	.status-bar.empty       { background: #2b2b2b; }
	.status-bar.bar-perfect { background: #22c55e; }
	.status-bar.bar-good    { background: #16a34a; }
	.status-bar.bar-degraded { background: #eab308; }
	.status-bar.bar-down    { background: #dc2626; }

	.status-indicator {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.375rem 0.75rem;
		border-radius: 0.25rem;
		font-size: 0.6875rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		white-space: nowrap;
		background: #2b2b2b;
	}

	.status-indicator.up   { color: #22c55e; }
	.status-indicator.down { color: #ef4444; }

	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: currentColor;
	}

	.status-indicator.up .status-dot {
		animation: pulse-dot 2s ease-in-out infinite;
	}

	@keyframes pulse-dot {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}

	.tooltip {
		display: none;
		position: absolute;
		background: #2b2b2b;
		color: #d1d5db;
		padding: 0.5rem 0.75rem;
		border-radius: 0.375rem;
		font-size: 0.75rem;
		white-space: pre-line;
		max-width: 200px;
		z-index: 50;
	}

	.tooltip-arrow {
		position: absolute;
		background: #2b2b2b;
		width: 8px;
		height: 8px;
		transform: rotate(45deg);
	}
</style>
