<script lang="ts">
	import { onMount, afterUpdate } from 'svelte';
	import { computePosition, autoPlacement, offset, shift, arrow } from '@floating-ui/dom';
	import type { PingResult } from '$lib/models';

	export let pingResults: PingResult[] = [];

	let status: 'up' | 'down' = 'down';
	let tooltipElement: HTMLDivElement;
	let arrowElement: HTMLDivElement;
	let currentTarget: HTMLElement | null = null;

	type AggregatedDay = {
		date: string;
		uptime: number;
		totalPings: number;
		successfulPings: number;
		timestamp: number;
	};

	let aggregatedDays: AggregatedDay[] = [];

	$: {
		// Only perform aggregation if we have more than 50 pings
		if (pingResults.length > 50) {
			const dailyPings = pingResults.reduce(
				(acc, ping) => {
					const date = new Date(ping.timestamp).toISOString().split('T')[0];
					if (!acc[date]) {
						acc[date] = {
							date,
							totalPings: 0,
							successfulPings: 0,
							timestamp: new Date(date).getTime() // Use start of day for consistent timestamp
						};
					}
					acc[date].totalPings++;
					if (!ping.error && ping.status_code < 400) {
						acc[date].successfulPings++;
					}
					return acc;
				},
				{} as Record<string, AggregatedDay>
			);

			aggregatedDays = Object.values(dailyPings)
				.map((day) => ({
					...day,
					uptime: (day.successfulPings / day.totalPings) * 100
				}))
				.sort((a, b) => a.timestamp - b.timestamp) // Sort ascending by date
				.slice(-50); // Take last 50 days
		} else {
			// If we have 50 or fewer pings, use them directly without aggregation
			aggregatedDays = pingResults
				.map((ping) => ({
					date: new Date(ping.timestamp).toISOString().split('T')[0],
					uptime: !ping.error && ping.status_code < 400 ? 100 : 0,
					totalPings: 1,
					successfulPings: !ping.error && ping.status_code < 400 ? 1 : 0,
					timestamp: ping.timestamp
				}))
				.sort((a, b) => a.timestamp - b.timestamp); // Sort ascending by date
		}
	}

	$: status = aggregatedDays.length > 0 && aggregatedDays[0].uptime >= 99.9 ? 'up' : 'down';

	const getStatusColor = (uptime: number) => {
		if (uptime >= 99.9) return 'bg-green-400';
		if (uptime >= 99.0) return 'bg-green-500';
		if (uptime >= 95) return 'bg-green-700';
		return 'bg-red-600';
	};

	const getStatusClasses = (status: 'up' | 'down') => {
		return status === 'up'
			? 'bg-green-500/20 text-green-400 border border-green-500/30 status-pulse'
			: 'bg-red-500/20 text-red-400 border border-red-500/30';
	};

	const formatTooltipContent = (day: AggregatedDay) => {
		const date = new Date(day.timestamp).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric'
		});
		return `${date} - Uptime: ${day.uptime.toFixed(2)}%\n${day.successfulPings}/${day.totalPings} successful pings`;
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
				autoPlacement({
					allowedPlacements: ['top', 'bottom']
				}),
				shift({ padding: 5 }),
				arrow({ element: arrowElement })
			]
		});

		Object.assign(tooltipElement.style, {
			left: `${x}px`,
			top: `${y}px`
		});

		// Handle arrow placement
		const { x: arrowX, y: arrowY } = middlewareData.arrow;
		const staticSide = {
			top: 'bottom',
			bottom: 'top'
		}[placement.split('-')[0]];

		Object.assign(arrowElement.style, {
			left: arrowX != null ? `${arrowX}px` : '',
			top: arrowY != null ? `${arrowY}px` : '',
			right: '',
			bottom: '',
			[staticSide]: '-4px'
		});
	};

	const hideTooltip = () => {
		tooltipElement.style.display = 'none';
		currentTarget = null;
	};

	// Cleanup on component unmount
	onMount(() => {
		return () => {
			hideTooltip();
		};
	});
</script>

<div class="flex justify-between items-center bg-[#202020] rounded-lg p-4">
	<div class="flex items-center gap-2">
		<div class="flex gap-1">
			{#if aggregatedDays.length < 50}
				{#each Array(50 - aggregatedDays.length) as _}
					<div class="w-2 h-6 rounded-sm bg-green-200/20"></div>
				{/each}
			{/if}
			{#each aggregatedDays as day}
				<div
					class={`w-2 h-6 rounded-sm transition-colors duration-200 cursor-pointer ${getStatusColor(day.uptime)}`}
					on:mouseenter={(e) => showTooltip(e, day)}
					on:mouseleave={hideTooltip}
				></div>
			{/each}
		</div>
	</div>

	<div class={`px-3 py-1 mx-2 ${getStatusClasses(status)} rounded-full text-sm font-medium`}>
		{status.toUpperCase()}
	</div>
</div>

<div class="tooltip" bind:this={tooltipElement}>
	<div class="tooltip-arrow" bind:this={arrowElement}></div>
</div>

<style>
	@keyframes pulse-ring {
		0% {
			transform: scale(0.95);
			box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.7);
		}
		70% {
			transform: scale(1);
			box-shadow: 0 0 0 10px rgba(34, 197, 94, 0);
		}
		100% {
			transform: scale(0.95);
			box-shadow: 0 0 0 0 rgba(34, 197, 94, 0);
		}
	}

	.status-pulse {
		position: relative;
		isolation: isolate;
		transform-origin: center;
		will-change: transform;
	}

	.status-pulse::before {
		content: '';
		position: absolute;
		inset: -4px;
		border-radius: 9999px;
		background: rgba(34, 197, 94, 0.7);
		animation: pulse-ring 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
		z-index: -1;
	}

	.tooltip {
		display: none;
		position: absolute;
		background: #2a2a2a;
		color: #999;
		padding: 8px 12px;
		border-radius: 6px;
		font-size: 0.9em;
		white-space: pre-line;
		max-width: 200px;
		z-index: 50;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
	}

	.tooltip-arrow {
		position: absolute;
		background: #2a2a2a;
		width: 8px;
		height: 8px;
		transform: rotate(45deg);
	}
</style>
