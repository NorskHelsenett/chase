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

  <!-- Rows -->
  <div class="space-y-1">
    {#each sites as site}
      <MonitorRow data={site} />
    {/each}
  </div>
</div>