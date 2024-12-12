<script lang="ts">
  import MonitorRow from './MonitorRow.svelte';
  import type { Server } from '$lib/models';
	import { goto } from '$app/navigation';

  export let sites: Server[] = [];
</script>

<div class="bg-[#202020] rounded-lg p-4">
  <table class="w-full border-spacing-4">
    <thead>
      <tr class="text-gray-400 font-medium">
        <th class="text-left font-medium">Status</th>
        <th class="text-left font-medium w-[30%]">Domain</th>
        <th class="text-left font-medium">Header</th>
        <th class="text-left font-medium">Cert</th>
        <th class="text-left font-medium">Admin Risk</th>
        <th class="text-left font-medium">API Risk</th>
        <th class="text-left font-medium">IP</th>
        <th class="text-left font-medium">Uptime</th>
      </tr>
    </thead>
    <tbody>
      {#if sites.length === 0}
        {#each Array(5) as _}
        <tr class="hover:bg-[#2b2b2b] transition-colors duration-200 ease-in-out cursor-pointer rounded-lg">
          <td>
            <div class="h-6 w-[7em] bg-gray-700/50 rounded-full animate-pulse"></div>
          </td>
          <td>
            <div class="h-5 w-32 bg-gray-700/50 rounded animate-pulse"></div>
          </td>
          <td>
            <div class="h-5 w-6 bg-gray-700/50 rounded animate-pulse"></div>
          </td>
          <td>
            <div class="h-5 w-6 bg-gray-700/50 rounded animate-pulse"></div>
          </td>
          <td>
            <div class="h-7 w-[7em] bg-gray-700/50 rounded-full animate-pulse"></div>
          </td>
          <td>
            <div class="h-7 w-[7em] bg-gray-700/50 rounded-full animate-pulse"></div>
          </td>
          <td>
            <div class="h-5 w-24 bg-gray-700/50 rounded animate-pulse"></div>
          </td>
          <td>
            <div class="flex gap-1">
              {#each Array(10) as _}
                <div class="w-1 h-4 bg-gray-700/50 rounded-sm animate-pulse"></div>
              {/each}
            </div>
          </td>
        </tr>
        {/each}
      {:else}
        {#each sites as site}
          <tr class="group transition-colors duration-200 ease-in-out hover:bg-[#2b2b2b] cursor-pointer rounded-lg" on:click={() => goto(`/server/${site.ID}`)}>
            <MonitorRow server={site} />
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>