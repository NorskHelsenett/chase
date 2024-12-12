<script lang="ts">
  export let data: {
    status: 'up' | 'down';
    title: string;
    headerScore: 'A+' | 'A' | 'B' | 'C' | 'D' | 'F';
    certScore: 'A+' | 'A' | 'B' | 'C' | 'D' | 'F';
    adminRisk: 'critical' | 'high' | 'medium' | 'low';
    apiRisk: 'critical' | 'high' | 'medium' | 'low';
    ip: string;
    uptime: Array<-1 | 0 | 1>;;
  };

  const getRiskColor = (risk: string) => {
    switch(risk) {
      case 'critical': return 'text-red-500 bg-red-500/20';
      case 'high': return 'text-orange-500 bg-orange-500/20';
      case 'medium': return 'text-yellow-500 bg-yellow-500/20';
      case 'low': return 'text-green-500 bg-green-500/20';
      default: return 'text-gray-500 bg-gray-500/20';
    }
  };

  const getScoreColor = (score: string) => {
    switch(score) {
      case 'A+':
      case 'A': return 'text-green-500';
      case 'B': return 'text-blue-500';
      case 'C': return 'text-yellow-500';
      case 'D':
      case 'F': return 'text-red-500';
      default: return 'text-gray-500';
    }
  };

  const getStatusClasses = (status: 'up' | 'down') => {
    return status === 'up'
      ? 'bg-green-500/20 text-green-400 border border-green-500/30'
      : 'bg-red-500/20 text-red-400 border border-red-500/30';
  };

  const getUptimeColor = (status: number) => {
    switch(status) {
      case 1: return 'bg-lime-400'; // up
      case -1: return 'bg-red-900'; // down
      default: return 'bg-green-200/20'; // no data
    }
  };
</script>

<div class="grid grid-cols-8 gap-4 p-3 hover:bg-[#2b2b2b] rounded-lg items-center cursor-pointer">
  <div class="flex items-center gap-2 w-full">
    <span class={`px-2 py-0.5 text-xs w-[7em] text-center font-medium rounded-full ${getStatusClasses(data.status)}`}>
      {data.status.toUpperCase()}
    </span>
  </div>

  <div class="text-white">{data.title}</div>

  <div class={getScoreColor(data.headerScore)}>{data.headerScore}</div>

  <div class={getScoreColor(data.certScore)}>{data.certScore}</div>

  <div class={`px-2 py-1 w-[7em] text-center rounded-full text-sm ${getRiskColor(data.adminRisk)}`}>
    {data.adminRisk}
  </div>

  <div class={`px-2 py-1 w-[7em] text-center rounded-full text-sm ${getRiskColor(data.apiRisk)}`}>
    {data.apiRisk}
  </div>

  <div class="text-gray-400">{data.ip}</div>

  <div class="flex gap-1">
    {#each data.uptime as status}
    <div class={`w-1 h-4 rounded-sm ${getUptimeColor(status)}`}></div>
  {/each}
  </div>
</div>