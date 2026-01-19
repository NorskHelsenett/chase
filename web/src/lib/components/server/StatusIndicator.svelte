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

const pingSuccessful = (ping: PingResult) => ping.status_code > 0 && ping.status_code < 400;

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
					if (pingSuccessful(ping)) {
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
					uptime: pingSuccessful(ping) ? 100 : 0,
					totalPings: 1,
					successfulPings: pingSuccessful(ping) ? 1 : 0,
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

<div class="status-container">
	<div class="status-bars">
		{#if aggregatedDays.length < 50}
			{#each Array(50 - aggregatedDays.length) as _}
				<div class="status-bar empty"></div>
			{/each}
		{/if}
		{#each aggregatedDays as day}
			<div
				class="status-bar {getStatusColor(day.uptime)}"
				on:mouseenter={(e) => showTooltip(e, day)}
				on:mouseleave={hideTooltip}
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
		padding: 0.75rem 0;
	}

	.status-bars {
		display: flex;
		gap: 3px;
		flex: 1;
	}

	.status-bar {
		width: 100%;
		max-width: 8px;
		height: 28px;
		border-radius: 2px;
		transition: all 0.15s ease;
		cursor: pointer;
	}

	.status-bar:hover {
		transform: scaleY(1.15);
	}

	.status-bar.empty {
		background: rgba(255, 255, 255, 0.05);
	}

	.status-bar.bg-green-400 {
		background: linear-gradient(180deg, #4ade80 0%, #22c55e 100%);
		box-shadow: 0 0 8px rgba(74, 222, 128, 0.3);
	}

	.status-bar.bg-green-500 {
		background: linear-gradient(180deg, #22c55e 0%, #16a34a 100%);
		box-shadow: 0 0 6px rgba(34, 197, 94, 0.25);
	}

	.status-bar.bg-green-700 {
		background: linear-gradient(180deg, #15803d 0%, #166534 100%);
	}

	.status-bar.bg-red-600 {
		background: linear-gradient(180deg, #dc2626 0%, #b91c1c 100%);
		box-shadow: 0 0 8px rgba(220, 38, 38, 0.4);
	}

	.status-indicator {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.375rem 0.875rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		white-space: nowrap;
	}

	.status-indicator.up {
		background: rgba(74, 222, 128, 0.1);
		color: #4ade80;
		border: 1px solid rgba(74, 222, 128, 0.2);
	}

	.status-indicator.down {
		background: rgba(248, 113, 113, 0.1);
		color: #f87171;
		border: 1px solid rgba(248, 113, 113, 0.2);
	}

	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: currentColor;
	}

	.status-indicator.up .status-dot {
		animation: pulse-glow 2s ease-in-out infinite;
	}

	@keyframes pulse-glow {
		0%, 100% {
			opacity: 1;
			box-shadow: 0 0 0 0 currentColor;
		}
		50% {
			opacity: 0.7;
			box-shadow: 0 0 0 4px transparent;
		}
	}

	.tooltip {
		display: none;
		position: absolute;
		background: #1a1a1a;
		color: #9ca3af;
		padding: 0.625rem 0.875rem;
		border-radius: 0.5rem;
		font-size: 0.75rem;
		white-space: pre-line;
		max-width: 200px;
		z-index: 50;
		border: 1px solid #333;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
	}

	.tooltip-arrow {
		position: absolute;
		background: #1a1a1a;
		width: 8px;
		height: 8px;
		transform: rotate(45deg);
		border: 1px solid #333;
		border-top: none;
		border-left: none;
	}
</style>
