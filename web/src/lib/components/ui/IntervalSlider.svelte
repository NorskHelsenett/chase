<!-- IntervalSlider.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();

  export let value = 5;
  export let label = 'Update Interval';

  const intervals = [
    { value: 5, label: '5m' },
    { value: 10, label: '10m' },
    { value: 15, label: '15m' },
    { value: 30, label: '30m' },
    { value: 60, label: '1h' },
    { value: 180, label: '3h' },
    { value: 360, label: '6h' },
    { value: 720, label: '12h' },
    { value: 10080, label: '1w' },
    { value: 20160, label: '14d' },
    { value: 43200, label: '30d' }
  ];

  let currentStepIndex = intervals.findIndex(interval => interval.value === value);
  if (currentStepIndex === -1) currentStepIndex = 0;

  function handleClick(index: number) {
    currentStepIndex = index;
    dispatch('change', intervals[index].value);
  }

  $: displayValue = intervals[currentStepIndex].label;
</script>

<div class="space-y-2">
  <div class="flex justify-between items-center">
    <span class="text-gray-300 text-sm">{label}</span>
    <span class="text-green-500 text-sm font-medium">{displayValue}</span>
  </div>

  <div class="relative">
    <!-- Track background -->
    <div class="h-1.5 bg-[#2b2b2b] rounded-full" style="margin-left:1%;width: 98% !important;">
      <!-- Active track -->
      <div
        class="absolute h-1.5 bg-green-600 rounded-full transition-all duration-200"
        style="width: {(currentStepIndex / (intervals.length - 1)) * 98}%"
      />
    </div>

    <!-- Step markers -->
    <div class="absolute inset-x-0 top-1/2 -translate-y-1/2 flex justify-between">
      {#each intervals as interval, i}
        <button
          class="group relative w-4 h-4 -ml-2 first:ml-0 last:mr-0 focus:outline-none"
          on:click={() => handleClick(i)}
          type="button"
        >
          <!-- Marker dot -->
          <div
            class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-4 h-4
            rounded-full transition-all duration-200 group-hover:bg-[#333] group-focus:ring-2
            group-focus:ring-green-500 group-focus:ring-opacity-50"
          >
            <div
              class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-2.5 h-2.5
              rounded-full transition-all duration-200
              {i <= currentStepIndex ? 'bg-green-600' : 'bg-[#2b2b2b] group-hover:bg-[#333]'}"
            />
          </div>

          <!-- Tooltip -->
          <div
            class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 py-1 px-2
            bg-[#2b2b2b] text-gray-200 text-xs rounded opacity-0 group-hover:opacity-100
            transition-opacity whitespace-nowrap pointer-events-none"
          >
            {interval.label}
          </div>
        </button>
      {/each}
    </div>
  </div>
</div>