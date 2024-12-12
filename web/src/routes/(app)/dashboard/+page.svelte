<script lang="ts">
  import { onMount } from 'svelte';
	import MonitorStats from "$lib/components/dashboard/MonitorStats.svelte";
  import MonitorTable from "$lib/components/dashboard/MonitorTable.svelte";
	import type { Server, Stats } from '$lib/models';

  let servers: Server[] = [];
  let stats: Stats = {
    up: 0,
    down: 0,
    criticalRisks: 0,
    highRisks: 0
  };

  onMount(async () => {
    try {
      const response = await fetch('/api/servers');
      servers = await response.json();

      stats = servers.reduce((acc: Stats, server: Server) => {
        const latestPing = server.ping_results?.[0];

        if (latestPing) {
          // Check if server is up (matching expected status code)
          if (latestPing.status_code  === server.expected_status) {
            acc.up += 1;
          } else {
            acc.down += 1;
          }

          // Check for TLS/cert risks
          if (!latestPing.tls_valid) {
            acc.criticalRisks += 1;
          }

          // Check for cert expiry (high risk if expires in less than 30 days)
          const certExpiryDate = new Date(latestPing.cert_expiry_date);
          const daysUntilExpiry = Math.floor(
            (certExpiryDate.getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24)
          );

          if (daysUntilExpiry < 30 && daysUntilExpiry > 0) {
            acc.highRisks += 1;
          }
        } else {
          acc.down += 1;
        }

        return acc;
      }, {
        up: 0,
        down: 0,
        criticalRisks: 0,
        highRisks: 0
      });

    } catch (error) {
      console.error('Failed to fetch server data:', error);
    }
  });

</script>
<div class="p-4 min-h-screen w-full">
  <MonitorStats {stats} />
  <MonitorTable sites={servers}/>
</div>