<script lang="ts">
  import { onMount } from 'svelte';
  import { browser } from '$app/environment';

  export let data: Array<{ timestamp: Date; value: number }> = [];

  let chartElement: HTMLElement;
  let chart: any;
  let ApexCharts: any;

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
            speed: 300
          }
        },
        stroke: {
          curve: 'smooth',
          width: 2,
          colors: ['#4ade80']
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
              show: true
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
          max: Math.max(...data.map(d => d.value)) + 100,
          tickAmount: 6,
          labels: {
            style: {
              colors: '#666'
            }
          }
        },
        tooltip: {
          theme: 'dark',
          x: {
            format: 'HH:mm'
          }
        },
        series: [
          {
            name: 'Response Time',
            data: data.map(d => [new Date(d.timestamp).getTime(), d.value])
          }
        ]
      };

      chart = new ApexCharts(chartElement, options);
      await chart.render();
    }
  };

  $: if (chart && data) {
    chart.updateOptions({
      yaxis: {
        min: 0,
        max: Math.max(...data.map(d => d.value)) + 100,
        tickAmount: 6,
        labels: {
          style: {
            colors: '#666'
          }
        }
      },
      series: [{
        name: 'Response Time',
        data: data.map(d => [new Date(d.timestamp).getTime(), d.value])
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

<div class="bg-[#202020] rounded-lg p-4">
  {#if browser}
    <div bind:this={chartElement}></div>
  {:else}
    <div class="h-[400px] flex items-center justify-center text-gray-400">
      Loading chart...
    </div>
  {/if}
</div>