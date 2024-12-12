<script lang="ts">
  import type { PingResult } from '$lib/models';

  export let pingResults: PingResult[] = [];
  let status: 'up' | 'down' = 'up';
  let sortedPingResults: PingResult[] = [];

  $: if(pingResults){
    sortedPingResults = [...pingResults].sort((a, b) => b.timestamp - a.timestamp);
    status = sortedPingResults[0]?.error || sortedPingResults[0]?.status_code >= 400 ? 'down' : 'up';
  }

  const getStatusColor = (ping: PingResult) => {
    if (ping.error || ping.status_code >= 400) {
      return 'bg-red-900';
    }
    return 'bg-green-400';
  };

  const getStatusClasses = (status: 'up' | 'down') => {
    return status === 'up'
      ? 'bg-green-500/20 text-green-400 border border-green-500/30'
      : 'bg-red-500/20 text-red-400 border border-red-500/30';
  };
</script>

<div class="flex justify-between items-center bg-[#202020] rounded-lg p-4">
  <div class="flex items-center gap-2">
    <div class="flex gap-1">
      {#each Array(50) as _, i}
        {#if i < (50 - sortedPingResults.length)}
          <div class="w-2 h-6 rounded-sm bg-green-200/20"></div>
        {:else}
          <div class={`w-2 h-6 rounded-sm ${getStatusColor(sortedPingResults[sortedPingResults.length - (i - (50 - sortedPingResults.length) + 1)])}`}></div>
        {/if}
      {/each}
    </div>
  </div>

  <div class={`px-3 py-1 mx-2 ${getStatusClasses(status)} rounded-full text-sm font-medium`}>
    {status.toUpperCase()}
  </div>
</div>