<script lang="ts">
  import { onMount } from 'svelte';
	import MonitorStats from "$lib/components/dashboard/MonitorStats.svelte";
  import MonitorTable from "$lib/components/dashboard/MonitorTable.svelte";

    // Types based on Go structs
    interface Server {
    ID: number;
    URL: string;
    Active: boolean;
    FollowRedirect: boolean;
    LastSuccess: string;
    FailureCount: number;
    NextCheck: string;
    AllowInsecure: boolean;
    PingResults: PingResult[];
  }

  interface PingResult {
    ID: number;
    ServerID: number;
    OrganizationName: string;
    StatusCode: number;
    IP: string;
    ResponseTime: number;
    Error: string;
    RedirectCount: number;
    Timestamp: string;
    TLSValid: boolean;
    CertExpiryDate: string;
    CertIssuer: string;
    CertCommonName: string;
    InsecureSkipVerify: boolean;
  }

  let pingResults: PingResult[] = []

  onMount(async () => {
    try {
      const response = await fetch('/api/servers');
      pingResults = await response.json();
      console.log(pingResults)
    } catch (error) {
      console.error('Failed to fetch server data:', error);
    }
  });

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
      uptime: [1,1,1,1,1,1,1,1,1,1]
    },
    {
      status: 'down',
      title: 'legacy.example.com',
      headerScore: 'D',
      certScore: 'C',
      adminRisk: 'critical',
      apiRisk: 'high',
      ip: '192.168.1.102',
      uptime: [-1,-1,-1,0,0,-1,-1,-1,-1,-1]
    },
    {
      status: 'up',
      title: 'blog.example.com',
      headerScore: '',
      certScore: 'A',
      adminRisk: '',
      apiRisk: '',
      ip: '192.168.1.103',
      uptime: [1,1,1,1,1,0,0,1,1,1]
    },
    {
      status: 'up',
      title: 'dev.example.com',
      headerScore: 'C',
      certScore: 'B',
      adminRisk: 'high',
      apiRisk: 'medium',
      ip: '192.168.1.104',
      uptime: [1,1,-1,-1,1,1,1,1,1,1]
    },
    {
      status: 'up',
      title: 'staging.example.com',
      headerScore: '',
      certScore: '',
      adminRisk: '',
      apiRisk: '',
      ip: '192.168.1.105',
      uptime: [1,1,1,1,1,1,1,1,1,1]
    },
    {
      status: 'down',
      title: 'test.example.com',
      headerScore: 'F',
      certScore: 'D',
      adminRisk: 'critical',
      apiRisk: 'critical',
      ip: '192.168.1.106',
      uptime: [-1,-1,-1,-1,0,0,-1,-1,-1,-1]
    },
    {
      status: 'up',
      title: 'mail.example.com',
      headerScore: 'A',
      certScore: 'A+',
      adminRisk: 'low',
      apiRisk: 'low',
      ip: '192.168.1.107',
      uptime: [1,1,1,1,1,1,1,1,1,1]
    }
];
</script>
<div class="p-4 min-h-screen w-full">
  <MonitorStats {stats} />
  <MonitorTable {sites}/>
</div>