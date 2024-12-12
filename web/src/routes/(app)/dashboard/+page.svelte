<script>
	import MonitorStats from "$lib/components/dashboard/MonitorStats.svelte";
  import MonitorTable from "$lib/components/dashboard/MonitorTable.svelte";
  const stats = {
      up: 5,
      down: 1,
      headerScoreAvg: 'A',
      certScoreAvg: 'A',
      criticalRisks: 1,
      highRisks: 1
    };

  const sites = [
    {
      status: 'up',
      title: 'api.example.com',
      headerScore: 'A+',
      certScore: 'A',
      adminRisk: 'low',
      apiRisk: 'low',
      ip: '192.168.1.100',
      uptime: [1,1,0,0,1,-1,-1,1,1,1]
    },
    {
      status: 'up',
      title: 'admin.example.com',
      headerScore: 'B',
      certScore: 'A',
      adminRisk: 'medium',
      apiRisk: 'low',
      ip: '192.168.1.101',
      uptime: [1,1,0,0,1,-1,-1,1,1,1]
    },
    {
      status: 'down',
      title: 'legacy.example.com',
      headerScore: 'D',
      certScore: 'C',
      adminRisk: 'critical',
      apiRisk: 'high',
      ip: '192.168.1.102',
      uptime: [1,1,0,0,1,-1,-1,1,1,1]
    }
  ];
</script>
<div class="p-4 min-h-screen w-full">
  <MonitorStats {stats} />
  <MonitorTable {sites}/>
</div>