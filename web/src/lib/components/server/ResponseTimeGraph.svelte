<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';

	export let data: Array<{ timestamp: Date; value: number }> = [];

	let chartElement: HTMLElement;
	let chart: any;
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
					background: '#141414',
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
					width: 3, // Slightly thicker line
					colors: ['#4ade80'],
					lineCap: 'round',
					// Added curve smoothness
					curve: 'smooth',
					smoothing: 0.35
				},
				fill: {
					type: 'gradient',
					gradient: {
						shadeIntensity: 1,
						opacityFrom: 0.25, // Slightly increased opacity
						opacityTo: 0.05, // Added slight opacity at the end
						stops: [0, 95, 100],
						colorStops: [
							{
								offset: 0,
								color: '#4ade80',
								opacity: 0.25
							},
							{
								offset: 95,
								color: '#4ade80',
								opacity: 0.05
							},
							{
								offset: 100,
								color: '#4ade80',
								opacity: 0
							}
						]
					}
				},
				grid: {
					borderColor: '#333',
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
							colors: '#666'
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
							colors: '#666'
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

	$: if (chart && data) {
		const smoothedData = calculateMovingAverage(data);
		chart.updateOptions({
			yaxis: {
				min: 0,
				max: Math.max(...data.map((d) => d.value)) + 5,
				tickAmount: 6,
				labels: {
					style: {
						colors: '#666'
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
		height: 400px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.loading-spinner {
		width: 1.5rem;
		height: 1.5rem;
		border: 2px solid #333;
		border-top-color: #4ade80;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	:global(.apexcharts-tooltip-box) {
		background: #1a1a1a !important;
		padding: 0.625rem 0.875rem !important;
		border-radius: 0.5rem;
		border: 1px solid #333 !important;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4) !important;
	}

	:global(.apexcharts-tooltip-box .timestamp) {
		color: #6b7280;
		font-size: 0.75rem;
		margin-bottom: 0.25rem;
	}

	:global(.apexcharts-tooltip-box .value) {
		color: #4ade80;
		font-weight: 600;
		font-size: 0.875rem;
	}

	:global(.apexcharts-tooltip-box .average) {
		color: #4ade80;
		font-weight: 500;
		margin-bottom: 2px;
	}

	:global(.apexcharts-tooltip-box .range) {
		color: #6b7280;
		font-size: 0.75rem;
	}
</style>
