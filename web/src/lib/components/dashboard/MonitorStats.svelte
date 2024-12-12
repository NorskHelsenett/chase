<script lang="ts">
  export let stats: {
    up: number;
    down: number;
    headerScoreAvg: string;
    certScoreAvg: string;
    criticalRisks: number;
    highRisks: number;
  } | null = null;
</script>

<div class="bg-[#202020] rounded-lg p-4 mb-4">
  <div class="grid grid-cols-6 gap-8">
    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Online</div>
      {#if stats}
        <div class="text-green-400 text-2xl font-semibold">{stats.up}</div>
      {:else}
        <div class="h-8 w-12 mx-auto bg-gray-700/50 rounded animate-pulse"></div>
      {/if}
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Offline</div>
      {#if stats}
        <div class="text-red-400 text-2xl font-semibold">{stats.down}</div>
      {:else}
        <div class="h-8 w-12 mx-auto bg-gray-700/50 rounded animate-pulse"></div>
      {/if}
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Avg Header</div>
      {#if stats}
        <div class="text-blue-400 text-2xl font-semibold">{stats.headerScoreAvg}</div>
      {:else}
        <div class="h-8 w-12 mx-auto bg-gray-700/50 rounded animate-pulse"></div>
      {/if}
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Avg Cert</div>
      {#if stats}
        <div class="text-blue-400 text-2xl font-semibold">{stats.certScoreAvg}</div>
      {:else}
        <div class="h-8 w-12 mx-auto bg-gray-700/50 rounded animate-pulse"></div>
      {/if}
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Critical Risks</div>
      {#if stats}
        <div class="text-red-400 text-2xl font-semibold">{stats.criticalRisks}</div>
      {:else}
        <div class="h-8 w-12 mx-auto bg-gray-700/50 rounded animate-pulse"></div>
      {/if}
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">High Risks</div>
      {#if stats}
        <div class="text-orange-400 text-2xl font-semibold">{stats.highRisks}</div>
      {:else}
        <div class="h-8 w-12 mx-auto bg-gray-700/50 rounded animate-pulse"></div>
      {/if}
    </div>
  </div>
</div>