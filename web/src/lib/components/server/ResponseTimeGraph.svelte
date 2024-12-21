<script lang="ts">
  import { onMount } from 'svelte';
  import { browser } from '$app/environment';

  export let data: Array<{ timestamp: Date; value: number }> = [];

  let chartElement: HTMLElement;
  let chart: any;
  let ApexCharts: any;

  // Calculate moving average
  const calculateMovingAverage = (data: Array<{ timestamp: Date; value: number }>, window: number = 5) => {
    return data.map((point, index) => {
      const start = Math.max(0, index - Math.floor(window / 2));
      const end = Math.min(data.length, index + Math.floor(window / 2) + 1);
      const values = data.slice(start, end).map(p => p.value);
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
          height: 400,
          toolbar: {
            show: false
          },
          background: '#202020',
          animations: {
            enabled: true,
            easing: 'linear',
            speed: 300,
            dynamicAnimation: {
              speed: 350
            }
          }
        },
        stroke: {
          curve: 'smooth',
          width: 2,
          colors: ['#4ade80'],
          lineCap: 'round'
        },
        fill: {
          type: 'gradient',
          gradient: {
            shadeIntensity: 1,
            opacityFrom: 0.2,
            opacityTo: 0,
            stops: [0, 90, 100],
            colorStops: [
              {
                offset: 0,
                color: '#4ade80',
                opacity: 0.2
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
          }
        },
        yaxis: {
          min: 0,
          max: Math.max(...data.map(d => d.value)) + 50,
          tickAmount: 6,
          labels: {
            style: {
              colors: '#666'
            }
          }
        },
        tooltip: {
          theme: 'dark',
          shared: true,
          custom: function({ series, seriesIndex, dataPointIndex }: any) {
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
        max: Math.max(...data.map(d => d.value)) + 50,
        tickAmount: 6,
        labels: {
          style: {
            colors: '#666'
          }
        }
      },
      series: [{
        name: 'Response Time',
        data: smoothedData
      }]
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

<style>
    :global(.apexcharts-tooltip-box) {
    background: #2a2a2a !important;
    /* border: 1px solid #3a3a3a !important; */
    padding: 8px 12px !important;
    border-radius: 6px;
  }

  :global(.apexcharts-tooltip-box .timestamp) {
    color: #999;
    font-size: 0.9em;
    margin-bottom: 4px;
  }

  :global(.apexcharts-tooltip-box .average) {
    color: #4ade80;
    font-weight: 500;
    margin-bottom: 2px;
  }

  :global(.apexcharts-tooltip-box .range) {
    color: #888;
    font-size: 0.9em;
  }
</style>

<div class="bg-[#202020] rounded-lg p-4">
  {#if browser}
    <div bind:this={chartElement}></div>
  {:else}
    <div class="h-[400px] flex items-center justify-center text-gray-400">
      Loading chart...
    </div>
  {/if}
</div>
