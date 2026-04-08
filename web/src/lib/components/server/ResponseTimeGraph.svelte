<script lang="ts">
	import { run } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { browser } from '$app/environment';

	interface Props {
		data?: Array<{ timestamp: Date; value: number }>;
	}

	let { data = [] }: Props = $props();

	let chartElement: HTMLElement = $state();
	let chart: any = $state();
	let ApexCharts: any;

	// Increased window size for smoother moving average
	const calculateMovingAverage = (
		data: Array<{ timestamp: Date; value: number }>,
		window: number = 1.5
	) => {
		return data.map((point, index) => {
			const start = Math.max(0, index - Math.floor(window / 2));
			const end = Math.min(data.length, index + Math.floor(window / 2) + 1);
			const values = data.slice(start, end).map((p) => p.value);
			const avg = values.reduce((sum, val) => sum + val, 0) / values.length;
			return [new Date(point.timestamp).getTime(), Math.round(avg)];
		});
	};

	const initChart = async () => {
		if (browser) {
			ApexCharts = (await import('apexcharts')).default;

			const options = {
				chart: {
					type: 'area',
					height: 350,
					toolbar: {
						show: false
					},
					background: '#202020',
					animations: {
						enabled: true,
						easing: 'cubicBezier', // Changed to cubicBezier for smoother animation
						speed: 800, // Increased animation duration
						dynamicAnimation: {
							speed: 650
						}
					}
				},
				stroke: {
					width: 2,
					colors: ['#22c55e'],
					lineCap: 'round',
					curve: 'smooth',
					smoothing: 0.35
				},
				fill: {
					type: 'solid',
					colors: ['#22c55e'],
					opacity: 0.08
				},
				grid: {
					borderColor: '#2b2b2b',
					strokeDashArray: 3,
					xaxis: {
						lines: {
							show: false
						}
					},
					yaxis: {
						lines: {
							show: false
						}
					}
				},
				dataLabels: {
					enabled: false
				},
				xaxis: {
					type: 'datetime',
					labels: {
						style: {
							colors: '#9ca3af'
						},
						datetimeFormatter: {
							hour: 'HH:mm'
						}
					},
					axisBorder: {
						show: false
					},
					axisTicks: {
						show: false
					},
					// Added tooltip animation
					tooltip: {
						enabled: true,
						animate: true,
						animateGradually: {
							enabled: true,
							delay: 150
						}
					}
				},
				yaxis: {
					min: 0,
					max: Math.max(...data.map((d) => d.value)) + 50,
					tickAmount: 6,
					labels: {
						style: {
							colors: '#9ca3af'
						},
						formatter: (value: number) => Math.round(value)
					}
				},
				tooltip: {
					theme: 'dark',
					shared: true,
					intersect: false, // Prevents tooltip from flickering
					custom: function ({ series, seriesIndex, dataPointIndex }: any) {
						const timestamp = new Date(data[dataPointIndex].timestamp);
						const value = series[0][dataPointIndex];
						return `
              <div class="apexcharts-tooltip-box">
                <div class="timestamp">${timestamp.toLocaleTimeString()}</div>
                <div class="value">Response Time: ${Math.round(value)}ms</div>
              </div>
            `;
					}
				},
				series: [
					{
						name: 'Response Time',
						data: calculateMovingAverage(data)
					}
				]
			};

			chart = new ApexCharts(chartElement, options);
			await chart.render();
		}
	};

	run(() => {
		if (chart && data) {
			const smoothedData = calculateMovingAverage(data);
			chart.updateOptions({
				yaxis: {
					min: 0,
					max: Math.max(...data.map((d) => d.value)) + 5,
					tickAmount: 6,
					labels: {
						style: {
							colors: '#9ca3af'
						}
					}
				},
				series: [
					{
						name: 'Response Time',
						data: smoothedData
					}
				]
			});
		}
	});

	onMount(async () => {
		await initChart();

		return () => {
			if (chart) {
				chart.destroy();
			}
		};
	});
</script>

<div class="graph-wrapper">
	{#if browser}
		<div bind:this={chartElement}></div>
	{:else}
		<div class="graph-loading">
			<div class="loading-spinner"></div>
			<span>Loading chart...</span>
		</div>
	{/if}
</div>

<style>
	.graph-wrapper {
		background: transparent;
	}

	.graph-loading {
		height: 350px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		color: #9ca3af;
		font-size: 0.875rem;
	}

	.loading-spinner {
		width: 1.5rem;
		height: 1.5rem;
		border: 2px solid #2b2b2b;
		border-top-color: #22c55e;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	:global(.apexcharts-tooltip-box) {
		background: #2b2b2b !important;
		padding: 0.5rem 0.75rem !important;
		border-radius: 0.375rem;
	}

	:global(.apexcharts-tooltip-box .timestamp) {
		color: #9ca3af;
		font-size: 0.75rem;
		margin-bottom: 0.25rem;
	}

	:global(.apexcharts-tooltip-box .value) {
		color: #22c55e;
		font-weight: 600;
		font-size: 0.875rem;
	}

	:global(.apexcharts-tooltip-box .average) {
		color: #22c55e;
		font-weight: 500;
		margin-bottom: 2px;
	}

	:global(.apexcharts-tooltip-box .range) {
		color: #9ca3af;
		font-size: 0.75rem;
	}
</style>
