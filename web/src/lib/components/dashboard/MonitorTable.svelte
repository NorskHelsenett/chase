<script lang="ts">
  import MonitorRow from './MonitorRow.svelte';

  export let sites = [];
</script>

<div class="bg-[#202020] rounded-lg p-4">
  <!-- Header -->
  <div class="grid grid-cols-8 gap-4 p-3 border-b border-gray-800 mb-2 text-gray-400 font-medium">
    <div>Status</div>
    <div>Domain</div>
    <div>Header</div>
    <div>Cert</div>
    <div>Admin Risk</div>
    <div>API Risk</div>
    <div>IP</div>
    <div>Uptime</div>
  </div>

  {#if sites.length === 0}
  {#each Array(5) as _}
    <div class="grid grid-cols-8 gap-4 p-3 hover:bg-[#2b2b2b] rounded-lg items-center">
      <div class="flex items-center gap-2 w-full">
        <div class="h-6 w-[7em] bg-gray-700/50 rounded-full animate-pulse"></div>
      </div>

      <div class="h-5 w-32 bg-gray-700/50 rounded animate-pulse"></div>

      <div class="h-5 w-6 bg-gray-700/50 rounded animate-pulse"></div>

      <div class="h-5 w-6 bg-gray-700/50 rounded animate-pulse"></div>

      <div class="h-7 w-[7em] bg-gray-700/50 rounded-full animate-pulse"></div>

      <div class="h-7 w-[7em] bg-gray-700/50 rounded-full animate-pulse"></div>

      <div class="h-5 w-24 bg-gray-700/50 rounded animate-pulse"></div>

      <div class="flex gap-1">
        {#each Array(10) as _}
          <div class="w-1 h-4 bg-gray-700/50 rounded-sm animate-pulse"></div>
        {/each}
      </div>
    </div>
  {/each}
{:else}
  <!-- Normal rows when data is available -->
  {#each sites as site}
    <MonitorRow data={site} />
  {/each}
{/if}
</div>