<!-- SecurityDashboard.svelte -->
<script lang="ts">
  import { Lock, Wrench, AlertTriangle, Server, Globe, Shield, File, Book, Key, List, Bug, CheckCircle, XCircle, Settings } from 'lucide-svelte';
  import { fade } from 'svelte/transition';
  import { onMount } from 'svelte';

  export let domain: string = '';

  let loading = true;
  let results: any = null;
  let error: string | null = null;

  function getRiskColor(risk: string): string {
    switch (risk?.toLowerCase()) {
      case 'critical':
        return 'text-red-600';
      case 'high':
        return 'text-red-500';
      case 'medium':
			case 'b':
			case 'b+':
        return 'text-yellow-500';
      case 'low':
			case 'a':
			case 'a+':
        return 'text-green-500';
      default:
        return 'text-gray-500';
    }
  }

  async function performScan() {
    loading = true;
    error = null;

    try {
      const response = await fetch(`/api/security/${encodeURIComponent(domain)}`);
      if (!response.ok) throw new Error('Scan failed');
      results = await response.json();
    } catch (err) {
      error = err.message;
      console.error('Scan failed:', err);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    performScan();
  });

  // Components
  const StatCard = ({ title, value, loading }) => {
    const skeleton = loading ? 'animate-pulse bg-gray-700/20' : '';
    return `
      <div class="text-center p-4 bg-gray-800/50 rounded-lg">
        <div class="text-2xl font-bold ${skeleton}">${value}</div>
        <div class="text-sm text-gray-400">${title}</div>
      </div>
    `;
  };

  const InfoCard = ({ label, value }) => {
    return `
      <div>
        <div class="text-gray-400 mb-1">${label}</div>
        <div>${value}</div>
      </div>
    `;
  };

	function getHighestRisk(headers) {
		let maxRiskIssue = null;
		if (headers?.issues) {
			for (let issue of headers.issues) {
				if (!maxRiskIssue || riskLevel[issue.risk] > riskLevel[maxRiskIssue.risk]) {
					maxRiskIssue = issue;
				}
			}
		}
		return maxRiskIssue
	}
</script>


<section class="mb-8">
	{#if loading}
		<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
			<div class="h-48 bg-gray-700 rounded-lg w-full mb-4"></div>
			<div class="space-y-2">
				{#each Array(3) as _}
					<div class="h-4 bg-gray-700 rounded w-full"></div>
				{/each}
			</div>
		</div>
	{:else if results}
		<div class="bg-[#202020] rounded-lg overflow-hidden" in:fade={{ duration: 200 }}>
			<!-- Website Screenshot -->
			<div class="w-full h-48 bg-[#2b2b2b] relative">
				<img
					src="/api/placeholder/800/400"
					alt="Website preview"
					class="w-full h-full object-cover opacity-80"
				/>
				<div class="absolute bottom-0 left-0 right-0 h-16"></div>
			</div>

			<!-- Summary Content -->
			<div class="p-6">
				<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
					<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
						<div
							class="text-2xl font-bold mb-1 {getRiskColor(results.headers.score)}"
						>
							{results.headers.score}
						</div>
						<div class="text-sm text-gray-400">Headers</div>
					</div>
					<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
						<div
							class="text-2xl font-bold mb-1 {getRiskColor(results.certificate.grade)}"
						>
							{results.certificate.grade}
						</div>
						<div class="text-sm text-gray-400">SSL/TLS</div>
					</div>
					<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
						<div class="text-2xl font-bold mb-1 uppercase {getRiskColor(results.adminPages.risk)}">
							{results.adminPages.risk}
						</div>
						<div class="text-sm text-gray-400">Admin Risk</div>
					</div>
					<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
						<div class="text-2xl font-bold mb-1 uppercase {getRiskColor(results.swagger.risk)}">
							{results.swagger.risk}
						</div>
						<div class="text-sm text-gray-400">API Risk</div>
					</div>
				</div>

				<div class="space-y-2">
					<div class="text-gray-400">Key Findings:</div>
					<ul class="text-gray-300 space-y-1">
						{#if getHighestRisk(results.headers)}
						<li>{getHighestRisk(results.headers)?.description || ""}
						{/if}
						<li>• Certificate valid until {results.certificate.validUntil}</li>
						<li>• {results.adminPages.exposed.length} admin pages exposed</li>
						<li>• {results.swagger.exposed
								? 'API documentation publicly accessible'
								: 'No public API documentation'}
						</li>
					</ul>
				</div>
			</div>
		</div>
	{/if}
</section>

<div class="max-w-3xl mx-auto p-4 space-y-6">
	<!-- Security Headers Section -->
	<section>
		<h2 class="text-xl flex items-center gap-2 mb-4">
			<Lock class="w-5 h-5" />
			Security Headers Analysis
		</h2>

		{#if loading}
			<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
				<div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
				<div class="space-y-2">
					{#each Array(3) as _}
						<div class="h-4 bg-gray-700 rounded w-full"></div>
					{/each}
				</div>
			</div>
		{:else if results}
			<div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
				<div class="flex items-center gap-4 mb-6">
					<div
						class="text-4xl font-bold {getRiskColor(results.headers.score)}"
					>
						{results.headers.score}
					</div>
					<div class="text-gray-400">Security Headers Score</div>
				</div>

				{#if results.headers.issues.length > 0}
					<div class="mb-4">
						<h3 class="text-red-400 flex items-center gap-2 mb-2">
							<AlertTriangle class="w-4 h-4" />
							Issues Found
						</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.headers.issues as issue}
								<li>• {issue.description}</li>
							{/each}
						</ul>
					</div>
				{/if}

				<div>
					<h3 class="text-green-400 mb-2">Passed Checks</h3>
					<ul class="space-y-1 text-gray-300">
						{#each results.headers.passed as check}
							<li>• {check}</li>
						{/each}
					</ul>
				</div>
			</div>
		{/if}
	</section>

	<!-- Certificate Section -->
	<section>
		<h2 class="text-xl flex items-center gap-2 mb-4">
			<Wrench class="w-5 h-5" />
			SSL/TLS Certificate Analysis
		</h2>

		{#if loading}
			<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
				<div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
				<div class="space-y-2">
					{#each Array(4) as _}
						<div class="h-4 bg-gray-700 rounded w-full"></div>
					{/each}
				</div>
			</div>
		{:else if results}
			<div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
				<div class="flex items-center gap-4 mb-6">
					<div
						class="text-4xl font-bold {getRiskColor(results.certificate.grade)}"
					>
						{results.certificate.grade}
					</div>
					<div class="text-gray-400">Certificate Grade</div>
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div>
						<h3 class="text-gray-400 mb-2">Certificate Details</h3>
						<p class="text-gray-300">Valid until: {results.certificate.validUntil}</p>
						<p class="text-gray-300">Issuer: {results.certificate.issuer}</p>
					</div>

					<div>
						<h3 class="text-gray-400 mb-2">Findings</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.certificate.findings as finding}
								<li>• {finding}</li>
							{/each}
						</ul>
					</div>
				</div>

				{#if results.certificate.warnings.length > 0}
					<div class="mt-4">
						<h3 class="text-yellow-400 flex items-center gap-2 mb-2">
							<AlertTriangle class="w-4 h-4" />
							Warnings
						</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.certificate.warnings as warning}
								<li>• {warning}</li>
							{/each}
						</ul>
					</div>
				{/if}
			</div>
		{/if}
	</section>

	<!-- Admin Pages Section -->
	<section>
		<h2 class="text-xl flex items-center gap-2 mb-4">
			<Server class="w-5 h-5" />
			Admin Pages Exposure
		</h2>

		{#if loading}
			<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
				<div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
				<div class="space-y-2">
					{#each Array(3) as _}
						<div class="h-4 bg-gray-700 rounded w-full"></div>
					{/each}
				</div>
			</div>
		{:else if results}
			<div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
				<div class="mb-4">
					<div class="text-lg font-semibold mb-2">
						Risk Level:
						<span class="{getRiskColor(results.adminPages.risk)} uppercase">{results.adminPages.risk}</span>
					</div>
				</div>

				<div class="mb-4">
					<h3 class="text-red-400 mb-2">Exposed Pages</h3>
					<ul class="space-y-1 text-gray-300">
						{#each results.adminPages.exposed as page}
							<li>• {page}</li>
						{/each}
					</ul>
				</div>

				<div>
					<h3 class="text-blue-400 mb-2">Recommendations</h3>
					<ul class="space-y-1 text-gray-300">
						{#each results.adminPages.recommendations as rec}
							<li>• {rec}</li>
						{/each}
					</ul>
				</div>
			</div>
		{/if}
	</section>

	<!-- Swagger/API Documentation Section -->
	<section>
		<h2 class="text-xl flex items-center gap-2 mb-4">
			<Book class="w-5 h-5" />
			API Documentation Exposure
		</h2>

		{#if loading}
			<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
				<div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
				<div class="space-y-2">
					{#each Array(3) as _}
						<div class="h-4 bg-gray-700 rounded w-full"></div>
					{/each}
				</div>
			</div>
		{:else if results}
			<div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
				<div class="mb-4">
					<div class="text-lg font-semibold mb-2">
						Risk Level:
						<span class="{getRiskColor(results.swagger.risk)} uppercase">{results.swagger.risk}</span>
					</div>
					<div class="text-gray-300">
						Status: {results.swagger.exposed
							? 'API Documentation Exposed'
							: 'No Public API Documentation Found'}
					</div>
				</div>

				{#if results.swagger.endpoints.length > 0}
					<div class="mb-4">
						<h3 class="text-red-400 mb-2">Exposed Endpoints</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.swagger.endpoints as endpoint}
								<li>• {endpoint}</li>
							{/each}
						</ul>
					</div>
				{/if}

				<div>
					<h3 class="text-blue-400 mb-2">Recommendations</h3>
					<ul class="space-y-1 text-gray-300">
						{#each results.swagger.recommendations as rec}
							<li>• {rec}</li>
						{/each}
					</ul>
				</div>
			</div>
		{/if}
	</section>
</div>

<div class="min-h-screen w-full max-w-7xl mx-auto p-4 space-y-6">
  {#if error}
    <div class="bg-red-500/10 border border-red-500/20 rounded-lg p-4 text-red-400" transition:fade>
      {error}
    </div>
  {/if}

  <!-- Infrastructure Analysis -->
  {#if !loading && results?.infrastructure}
    <div class="bg-[#202020] rounded-lg p-6" transition:fade>
      <div class="flex items-center gap-2 mb-6">
        <Server class="w-5 h-5" />
        <h2 class="text-xl">Infrastructure</h2>
      </div>

      <div class="space-y-6">
        <div class="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div>
            <div class="text-gray-400 mb-1">IP Address</div>
            <div>{results.infrastructure.ip}</div>
          </div>
          <div>
            <div class="text-gray-400 mb-1">HTTP Status</div>
            <div>{results.infrastructure.status}</div>
          </div>
          <div>
            <div class="text-gray-400 mb-1">Server</div>
            <div>{results.infrastructure.server || 'Not disclosed'}</div>
          </div>
        </div>

        {#if results.infrastructure.tech?.length > 0}
          <div>
            <h3 class="text-gray-400 mb-2">Technologies</h3>
            <div class="grid grid-cols-2 md:grid-cols-3 gap-2">
              {#each results.infrastructure.tech as tech}
                <div class="p-2 bg-gray-800/50 rounded-lg">
                  <div class="font-medium">{tech.name}</div>
                  {#if tech.version}
                    <div class="text-sm text-gray-400">v{tech.version}</div>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if results.infrastructure.findings?.length > 0}
          <div>
            <h3 class="text-yellow-400 mb-2">Findings</h3>
            <div class="space-y-4">
              {#each results.infrastructure.findings as finding}
                <div class={getRiskColor(finding.risk)}>
                  <div class="font-medium">{finding.description}</div>
                  {#if finding.evidence}
                    <div class="text-sm text-gray-400">Evidence: {finding.evidence}</div>
                  {/if}
                  {#if finding.mitigation}
                    <div class="text-sm text-blue-400">Mitigation: {finding.mitigation}</div>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Content Security -->
  {#if !loading && results?.contentSecurity}
    <div class="bg-[#202020] rounded-lg p-6" transition:fade>
      <div class="flex items-center gap-2 mb-6">
        <Shield class="w-5 h-5" />
        <h2 class="text-xl">Content Security Policy</h2>
      </div>

      <div class="space-y-4">
        <div>
          <div class="text-gray-400 mb-1">Policy Status</div>
          <div>{results.contentSecurity.hasPolicy ? 'Implemented' : 'Missing'}</div>
        </div>

        {#if results.contentSecurity.policyDetails}
          <div class="bg-gray-800/50 p-3 rounded-lg overflow-x-auto">
            <pre class="text-sm">{results.contentSecurity.policyDetails}</pre>
          </div>
        {/if}

        {#if results.contentSecurity.violations?.length > 0}
          <div>
            <h3 class="text-red-400 mb-2">Policy Violations</h3>
            <ul class="space-y-1">
              {#each results.contentSecurity.violations as violation}
                <li class="text-gray-300">• {violation}</li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- DNS Analysis -->
  {#if !loading && results?.dnsRecords}
    <div class="bg-[#202020] rounded-lg p-6" transition:fade>
      <div class="flex items-center gap-2 mb-6">
        <Globe class="w-5 h-5" />
        <h2 class="text-xl">DNS Records</h2>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        {#if results.dnsRecords.aRecords?.length > 0}
          <div>
            <h3 class="text-gray-400 mb-2">A Records</h3>
            <ul class="space-y-1">
              {#each results.dnsRecords.aRecords as record}
                <li class="text-gray-300">• {record}</li>
              {/each}
            </ul>
          </div>
        {/if}

        {#if results.dnsRecords.mxRecords?.length > 0}
          <div>
            <h3 class="text-gray-400 mb-2">MX Records</h3>
            <ul class="space-y-1">
              {#each results.dnsRecords.mxRecords as record}
                <li class="text-gray-300">• {record}</li>
              {/each}
            </ul>
          </div>
        {/if}

        {#if results.dnsRecords.txtRecords?.length > 0}
          <div>
            <h3 class="text-gray-400 mb-2">TXT Records</h3>
            <ul class="space-y-1">
              {#each results.dnsRecords.txtRecords as record}
                <li class="text-gray-300">• {record}</li>
              {/each}
            </ul>
          </div>
        {/if}

        {#if results.dnsRecords.findings?.length > 0}
          <div class="md:col-span-2">
            <h3 class="text-yellow-400 mb-2">Findings</h3>
            <ul class="space-y-1">
              {#each results.dnsRecords.findings as finding}
                <li class="text-gray-300">• {finding}</li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
    </div>
  {/if}

	<!-- File Exposure -->
  {#if !loading && results?.fileExposure}
    <div class="bg-[#202020] rounded-lg p-6" transition:fade>
      <div class="flex items-center gap-2 mb-6">
        <File class="w-5 h-5" />
        <h2 class="text-xl">Sensitive File Exposure</h2>
      </div>

      <div class="space-y-4">
        {#if results.fileExposure.exposedFiles?.length > 0}
          <div>
            <h3 class="text-red-400 mb-2">Exposed Files</h3>
            <div class="space-y-2">
              {#each results.fileExposure.exposedFiles as file}
                <div class="p-3 bg-gray-800/50 rounded-lg">
                  <div class={getRiskColor(file.risk)}>
                    <div class="font-medium">{file.path}</div>
                    <div class="text-sm text-gray-400">{file.description}</div>
                    <div class="text-xs text-gray-500">Type: {file.type}</div>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {:else}
          <div class="text-green-400">No sensitive files exposed</div>
        {/if}
      </div>
    </div>
  {/if}

	<!-- Add these new sections just before the Scan Errors section -->

<!-- Robots.txt Analysis -->
{#if !loading && results?.robotsTxt}
<div class="bg-[#202020] rounded-lg overflow-hidden">
	<div class="p-6">
		<div class="flex items-center gap-2 mb-6">
			<File class="w-5 h-5" />
			<h2 class="text-xl">robots.txt Analysis</h2>
		</div>

		<div class="space-y-4">
			<div class="grid grid-cols-2 gap-4">
				<div>
					<div class="text-gray-400 mb-1">Status</div>
					<div class={results.robotsTxt.exists ? 'text-green-400' : 'text-yellow-400'}>
						{results.robotsTxt.exists ? 'Found' : 'Not Found'}
					</div>
				</div>
				<div>
					<div class="text-gray-400 mb-1">Content Type</div>
					<div class={results.robotsTxt.contentType === 'text/plain' ? 'text-green-400' : 'text-red-400'}>
						{results.robotsTxt.contentType || 'N/A'}
					</div>
				</div>
			</div>

			{#if results.robotsTxt.exists && results.robotsTxt.contentType === 'text/plain'}
				<div>
					<h3 class="text-gray-400 mb-2">Content</h3>
					<pre class="p-3 bg-gray-800/50 rounded-lg overflow-x-auto text-sm whitespace-pre-wrap">{results.robotsTxt.content}</pre>
				</div>
			{/if}

			{#if results.robotsTxt.findings?.length > 0}
				<div>
					<h3 class="text-yellow-400 mb-2">Findings</h3>
					<div class="space-y-4">
						{#each results.robotsTxt.findings as finding}
							<div class={getRiskColor(finding.risk)}>
								<div class="font-medium">{finding.description}</div>
								{#if finding.evidence}
									<div class="text-sm text-gray-400">Evidence: {finding.evidence}</div>
								{/if}
								{#if finding.mitigation}
									<div class="text-sm text-blue-400">Mitigation: {finding.mitigation}</div>
								{/if}
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	</div>
</div>
{/if}

<!-- Security.txt Analysis -->
{#if !loading && results?.securityTxt}
  <div class="bg-[#202020] rounded-lg overflow-hidden">
    <div class="p-6">
      <div class="flex items-center gap-2 mb-6">
        <Shield class="w-5 h-5" />
        <h2 class="text-xl">security.txt Analysis</h2>
      </div>

      <div class="space-y-4">
        <div class="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div>
            <div class="text-gray-400 mb-1">Status</div>
            <div class={results.securityTxt.exists ? 'text-green-400' : 'text-yellow-400'}>
              {results.securityTxt.exists ? 'Found' : 'Not Found'}
            </div>
          </div>
          <div>
            <div class="text-gray-400 mb-1">Content Type</div>
            <div class={results.securityTxt.contentType === 'text/plain' ? 'text-green-400' : 'text-red-400'}>
              {results.securityTxt.contentType || 'N/A'}
            </div>
          </div>
          {#if results.securityTxt.exists}
            <div>
              <div class="text-gray-400 mb-1">PGP Signature</div>
              <div class={results.securityTxt.validSignature ? 'text-green-400' : 'text-yellow-400'}>
                {results.securityTxt.validSignature ? 'Signed' : 'Not Signed'}
              </div>
            </div>
          {/if}
        </div>

        {#if results.securityTxt.exists && results.securityTxt.contentType === 'text/plain'}
          {#if results.securityTxt.contacts?.length > 0}
            <div>
              <h3 class="text-gray-400 mb-2">Contact Points</h3>
              <div class="space-y-2">
                {#each results.securityTxt.contacts as contact}
                  <div class="p-2 bg-gray-800/50 rounded-lg flex items-center gap-2">
                    <span class="text-blue-400">
                      {#if contact.startsWith('mailto:')}
                        {contact.replace('mailto:', '')}
                      {:else}
                        {contact}
                      {/if}
                    </span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if results.securityTxt.expiration}
            {@const daysUntilExpiry = Math.floor((new Date(results.securityTxt.expiration) - new Date()) / (1000 * 60 * 60 * 24))}
            <div>
              <h3 class="text-gray-400 mb-2">Expiration</h3>
              <div class="p-2 bg-gray-800/50 rounded-lg">
                <div class="font-medium">
                  {new Date(results.securityTxt.expiration).toLocaleDateString()}
                </div>
                <div class={daysUntilExpiry < 30 ? 'text-red-400' : 
                           daysUntilExpiry < 90 ? 'text-yellow-400' : 'text-green-400'}>
                  {daysUntilExpiry} days remaining
                </div>
              </div>
            </div>
          {/if}

          {#if results.securityTxt.canonical?.length > 0}
            <div>
              <h3 class="text-gray-400 mb-2">Canonical URLs</h3>
              <div class="space-y-2">
                {#each results.securityTxt.canonical as url}
                  <div class="p-2 bg-gray-800/50 rounded-lg">
                    <span class="text-blue-400">{url}</span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if results.securityTxt.encryptions?.length > 0}
            <div>
              <h3 class="text-gray-400 mb-2">Encryption Keys</h3>
              <div class="space-y-2">
                {#each results.securityTxt.encryptions as key}
                  <div class="p-2 bg-gray-800/50 rounded-lg text-sm">
                    <span class="text-green-400">{key}</span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          <div>
            <h3 class="text-gray-400 mb-2">Content</h3>
            <pre class="p-3 bg-gray-800/50 rounded-lg overflow-x-auto text-sm whitespace-pre-wrap">{results.securityTxt.content}</pre>
          </div>
        {/if}

        {#if results.securityTxt.findings?.length > 0}
          <div>
            <h3 class="text-yellow-400 mb-2">Findings</h3>
            <div class="space-y-4">
              {#each results.securityTxt.findings as finding}
                <div class={getRiskColor(finding.risk)}>
                  <div class="font-medium">{finding.description}</div>
                  {#if finding.evidence}
                    <div class="text-sm text-gray-400">Evidence: {finding.evidence}</div>
                  {/if}
                  {#if finding.mitigation}
                    <div class="text-sm text-blue-400">Mitigation: {finding.mitigation}</div>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

  <!-- Scan Errors -->
  {#if !loading && results?.scanErrors?.length > 0}
    <div class="bg-[#202020] rounded-lg p-6" transition:fade>
      <div class="flex items-center gap-2 mb-6">
        <AlertTriangle class="w-5 h-5 text-yellow-500" />
        <h2 class="text-xl">Scan Issues</h2>
      </div>

      <div class="space-y-4">
        {#each results.scanErrors as error}
          <div class="p-3 bg-gray-800/50 rounded-lg">
            <div class="font-medium text-yellow-500">{error.component}</div>
            <div class="text-sm text-gray-400">{error.error}</div>
            <div class="text-xs text-gray-500">
              {new Date(error.timestamp).toLocaleString()}
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  @keyframes pulse {
    0%, 100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }

  .animate-pulse {
    animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
  }
</style>