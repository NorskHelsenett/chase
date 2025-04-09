<script lang="ts">
  import type { Server } from '$lib/models';

  type ServerRowData = {
    status: 'up' | 'down';
    title: string;
    headerScore: 'A+' | 'A' | 'B' | 'C' | 'D' | 'F' | '';
    certScore: 'A+' | 'A' | 'B' | 'C' | 'D' | 'F' | '';
    adminRisk: 'critical' | 'high' | 'medium' | 'low' | '';
    apiRisk: 'critical' | 'high' | 'medium' | 'low' | '';
    uptime: Array<-1 | 0 | 1>;
  };

  export let server: Server;

  let rowData: ServerRowData;

  $: rowData = mapServerToRowData(server);

  function mapServerToRowData(server: Server): ServerRowData {
    const sortedPings = [...(server.ping_results || [])].sort((a, b) =>
      new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    );

    return {
      status: !sortedPings.length || sortedPings[0]?.error || sortedPings[0]?.status_code >= 400 ? 'down' : 'up',
      title: server.url,
      headerScore: server.security?.headerRisk || '',
      certScore: server.security?.certRisk || '',
      adminRisk: server.security?.adminRisk || '',
      apiRisk: server.security?.apiRisk || '',
      uptime: (server.ping_results || [])
        .slice(0, 10)
        .reverse()
        .map(ping => ping.error || ping.status_code >= 400 ? -1 : 1)
    };
  }

  const getRiskColor = (risk: string) => {
    switch(risk?.toLowerCase()) {
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
      case 'B+':
      case 'B': return 'text-yellow-500';
      case 'C': return 'text-blue-500';
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
      case 1: return 'bg-lime-300/70'; // up
      case -1: return 'bg-red-500'; // down
      default: return 'bg-green-200/10'; // no data
    }
  };
</script>

<td class="py-2">
  <span class={`px-2 py-0.5 text-xs w-[7em] text-center font-medium rounded-full ${getStatusClasses(rowData.status)}`}>
    {rowData.status.toUpperCase()}
  </span>
</td>

<td class="text-white min-w-[300px]">
  <div class="whitespace-nowrap overflow-hidden text-ellipsis">
    {rowData.title}
  </div>
</td>

<td class={getScoreColor(rowData.headerScore)}>{rowData.headerScore}</td>

<td class={getScoreColor(rowData.certScore)}>{rowData.certScore}</td>

<td>
  <div class={`px-2 py-1 w-[7em] text-center rounded-full text-sm ${getRiskColor(rowData.adminRisk)}`}>
    {rowData.adminRisk || ''}
  </div>
</td>

<td>
  <div class={`px-2 py-1 w-[7em] text-center rounded-full text-sm ${getRiskColor(rowData.apiRisk)}`}>
    {rowData.apiRisk || ''}
  </div>
</td>

<td>
  <div class="flex gap-1">
    {#each Array(10) as _, i}
      {#if i < (10 - rowData.uptime.length)}
        <div class="w-1 h-4 rounded-sm bg-green-200/20"></div>
      {:else}
        <div class={`w-1 h-4 rounded-sm ${getUptimeColor(rowData.uptime[rowData.uptime.length - (i - (10 - rowData.uptime.length) + 1)])}`}></div>
      {/if}
    {/each}
  </div>
</td>