<script lang="ts">
  import MonitorRow from './MonitorRow.svelte';
  import type { Server } from '$lib/models';
  import { goto } from '$app/navigation';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let sites: Server[] = [];
  export let visibleServerIds = new Set<string>(); // Bind to parent component

  let sortField: keyof Server | 'status' | 'header' | 'cert' | 'adminRisk' | 'apiRisk' | 'uptime' | null = null;
  let sortDirection: 'asc' | 'desc' = 'asc';

  // Helper function to convert grade to numeric value for sorting
  function gradeToNumber(grade: string): number {
  const grades = {
    'A+': 7,
    'A': 6,
    'B+': 5,
    'B': 4,
    'C': 3,
    'D': 2,
    'F': 1,
    '': 0
  };
  return grades[grade as keyof typeof grades] || 0;
}

  // Helper function to convert risk level to numeric value
  function riskToNumber(risk: string): number {
  const risks = {
    'critical': 4,
    'high': 3,
    'medium': 2,
    'low': 1,
    '': 0
  };
  return risks[risk.toLowerCase() as keyof typeof risks] || 0;
}

  // Helper function to calculate uptime percentage from ping results
  function getUptimePercentage(server: Server): number {
    const pings = server.ping_results || [];
    if (pings.length === 0) return 0;

    const successfulPings = pings.filter(ping =>
      ping.status_code === server.expected_status
    ).length;

    return (successfulPings / pings.length) * 100;
  }

  function toggleSort(field: typeof sortField) {
    if (sortField === field) {
      sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
    } else {
      sortField = field;
      sortDirection = 'asc';
    }

    sites = [...sites].sort((a, b) => {
      let valueA, valueB;

      switch (field) {
        case 'status':
          valueA = getLatestPingStatus(a);
          valueB = getLatestPingStatus(b);
          break;
        case 'url':
          valueA = a.url.toLowerCase();
          valueB = b.url.toLowerCase();
          break;
        case 'header':
        case 'cert':
          valueA = gradeToNumber(getLatestGrade(a, field));
          valueB = gradeToNumber(getLatestGrade(b, field));
          break;
        case 'adminRisk':
        case 'apiRisk':
          valueA = riskToNumber(getLatestRisk(a, field));
          valueB = riskToNumber(getLatestRisk(b, field));
          break;
        case 'ip':
          valueA = a.ping_results?.[0]?.ip || '';
          valueB = b.ping_results?.[0]?.ip || '';
          break;
        case 'uptime':
          valueA = getUptimePercentage(a);
          valueB = getUptimePercentage(b);
          break;
        default:
          valueA = a[field as keyof Server];
          valueB = b[field as keyof Server];
      }

      if (valueA < valueB) return sortDirection === 'asc' ? -1 : 1;
      if (valueA > valueB) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });
  }

  function getLatestPingStatus(server: Server): boolean {
  const sortedPings = [...(server.ping_results || [])].sort((a, b) =>
    new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  );
  // Match your row logic
  return !sortedPings.length || sortedPings[0]?.error || sortedPings[0]?.status_code >= 400 ? false : true;
}

function getLatestGrade(server: Server, type: 'header' | 'cert'): string {
  // First try the new fields, fall back to security object if not available
  if (type === 'header') {
    return server.header_score || (server.security?.headerRisk || '');
  } else {
    return server.cert_score || (server.security?.certRisk || '');
  }
}

function getLatestRisk(server: Server, type: 'adminrisk' | 'apirisk'): string {
  // First try the new fields, fall back to security object if not available
  if (type === 'adminrisk') {
    // Convert risk levels to lowercase for consistent display
    return (server.admin_risk?.toLowerCase() || server.security?.adminRisk || '');
  } else {
    return (server.api_risk?.toLowerCase() || server.security?.apiRisk || '');
  }
}
</script>

<div class="bg-[#202020] rounded-lg p-4">
  <table class="w-full border-spacing-4">
    <thead>
      <tr class="text-gray-400 font-medium">
        <th
          class="text-left font-medium cursor-pointer hover:text-gray-200 transition-colors group"
          on:click={() => toggleSort('status')}
        >
          Status
          <span class="ml-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {sortField === 'status' ? (sortDirection === 'asc' ? '↑' : '↓') : '↕'}
          </span>
        </th>
        <th
          class="text-left font-medium w-[30%] cursor-pointer hover:text-gray-200 transition-colors group"
          on:click={() => toggleSort('url')}
        >
          Domain
          <span class="ml-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {sortField === 'url' ? (sortDirection === 'asc' ? '↑' : '↓') : '↕'}
          </span>
        </th>
        <th
          class="text-left font-medium cursor-pointer hover:text-gray-200 transition-colors group"
          on:click={() => toggleSort('header')}
        >
          Header
          <span class="ml-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {sortField === 'header' ? (sortDirection === 'asc' ? '↑' : '↓') : '↕'}
          </span>
        </th>
        <th
          class="text-left font-medium cursor-pointer hover:text-gray-200 transition-colors group"
          on:click={() => toggleSort('cert')}
        >
          Cert
          <span class="ml-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {sortField === 'cert' ? (sortDirection === 'asc' ? '↑' : '↓') : '↕'}
          </span>
        </th>
        <th
          class="text-left font-medium cursor-pointer hover:text-gray-200 transition-colors group"
          on:click={() => toggleSort('adminRisk')}
        >
          Admin Risk
          <span class="ml-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {sortField === 'adminRisk' ? (sortDirection === 'asc' ? '↑' : '↓') : '↕'}
          </span>
        </th>
        <th
          class="text-left font-medium cursor-pointer hover:text-gray-200 transition-colors group"
          on:click={() => toggleSort('apiRisk')}
        >
          API Risk
          <span class="ml-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {sortField === 'apiRisk' ? (sortDirection === 'asc' ? '↑' : '↓') : '↕'}
          </span>
        </th>
        <th
          class="text-left font-medium cursor-pointer hover:text-gray-200 transition-colors group"
          on:click={() => toggleSort('uptime')}
        >
          Uptime
          <span class="ml-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {sortField === 'uptime' ? (sortDirection === 'asc' ? '↑' : '↓') : '↕'}
          </span>
        </th>
      </tr>
    </thead>
    <tbody>
      {#if sites.length === 0}
        <tr>
          <td colspan="7" class="text-center py-8 text-gray-500">
            No servers found
          </td>
        </tr>
      {:else}
        {#each sites as site}
          <tr
            data-server-id={site.ID}
            class="group transition-colors duration-200 ease-in-out hover:bg-[#2b2b2b] cursor-pointer rounded-lg"
            on:click={() => goto(`/server/${site.ID}`)}
          >
            <MonitorRow server={site} />
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>