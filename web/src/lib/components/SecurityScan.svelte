<!-- SecurityDashboard.svelte -->
<script lang="ts">
	import { AlertTriangle, Server, Globe, Shield, File, Book } from 'lucide-svelte';
	import { fade } from 'svelte/transition';
	import { onMount } from 'svelte';
	import Certificate from './scan/Certificate.svelte';
	import Headers from './scan/Headers.svelte';

	export let domain: string = '';
	export let searchResults = {};

	let loading = true;
	let results: any = null;
	let error: string | null = null;

	function getRiskColor(value: string): string {
    switch (value?.toLowerCase()) {
        // Risk levels
        case 'critical':
        case 'f':
        case 'e':
            return 'text-red-500';
        case 'high':
        case 'd':
            return 'text-orange-500';
        case 'medium':
        case 'c':
            return 'text-yellow-500';
        case 'low':
        case 'b':
            return 'text-blue-500';
        case 'info':
        case 'a':
        case 'a+':
            return 'text-green-500';
        default:
            return 'text-gray-500';
    }
}

	async function performScan() {
		// If domain is empty but we have search results, just set the domain and return
		if (!domain && searchResults) {
			domain = extractDomain(searchResults.targetUrl);
			results = searchResults;
			loading = false;
			return;
		}

		// Only proceed with security scan if we had an original domain
		if (!domain) {
			return;
		}

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

	function extractDomain(url: string): string {
		if (!url) return '';
		return url.replace(/^(https?:\/\/)/, '').split('/')[0];
	}

	onMount(() => {
		performScan();
	});

	const riskLevel = {
		info: 0,
		low: 1,
		medium: 2,
		high: 3,
		critical: 4
	};

	function getHighestRisk(headers) {
		let maxRiskIssue = null;
		if (headers?.issues) {
			for (let issue of headers.issues) {
				if (
					!maxRiskIssue ||
					riskLevel[issue.risk.toLowerCase()] > riskLevel[maxRiskIssue.risk.toLowerCase()]
				) {
					maxRiskIssue = issue;
				}
			}
		}
		return maxRiskIssue;
	}

  $: hasScanErrors = !loading && results?.scanErrors?.length > 0;
</script>

{#if hasScanErrors}
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
{:else}
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
				<div class="w-full h-48 bg-[#2b2b2b] relative">
					{#if loading}
						<div class="absolute inset-0 animate-pulse bg-gray-700/50" />
					{:else}
						<div class="w-full h-48 bg-[#2b2b2b] relative">
							{#key domain}
								<div class="relative w-full h-full">
									<img
										src={`/api/screenshot/${encodeURIComponent(domain)}`}
										alt="Website preview"
										class="absolute w-full h-full object-cover opacity-0 transition-opacity duration-300"
										on:load={(e) => {
											e.target.classList.remove('opacity-0');
											e.target.classList.add('opacity-80');
											// Find and remove the loading overlay
											const overlay = e.target.parentElement.querySelector('.loading-overlay');
											if (overlay) overlay.remove();
										}}
										on:error={(e) => {
											// Remove the loading overlay on error too
											const overlay = e.target.parentElement.querySelector('.loading-overlay');
											if (overlay) overlay.remove();
										}}
									/>
									<div
										class="loading-overlay absolute inset-0 bg-gray-700/50 animate-pulse"
										style="animation-duration: 2s;"
									/>
								</div>
							{/key}
							<div class="absolute bottom-0 left-0 right-0 h-16" />
						</div>
					{/if}
					<div class="absolute bottom-0 left-0 right-0 h-16" />
				</div>
				<div class="absolute bottom-0 left-0 right-0 h-16"></div>
			</div>

			<!-- Summary Content -->
			<div class="p-6">
				<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
					<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
						<div class="text-2xl font-bold mb-1 {getRiskColor(results.headers.score)}">
							{results.headers.score}
						</div>
						<div class="text-sm text-gray-400">Headers</div>
					</div>
					<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
						<div class="text-2xl font-bold mb-1 {getRiskColor(results.certificate.grade)}">
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
							<li>{getHighestRisk(results.headers)?.description || ''}</li>{/if}
						<li>• Certificate valid until {results.certificate.validUntil}</li>
						<li>• {results.adminPages.exposed.length} admin pages exposed</li>
						<li>
							• {results.swagger.exposed
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
	<Headers {loading} {results} />

	<!-- Certificate Section -->
	<Certificate {loading} {results} />

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
						<span class="{getRiskColor(results.adminPages.risk)} uppercase"
							>{results.adminPages.risk}</span
						>
					</div>
				</div>

				{#if results.adminPages.exposed.length > 0}
					<div class="mb-4">
						<h3 class="text-red-400 mb-2">Exposed Pages</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.adminPages.exposed as page}
								<li>• {page}</li>
							{/each}
						</ul>
					</div>
				{/if}

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
						<span class="{getRiskColor(results.swagger.risk)} uppercase"
							>{results.swagger.risk}</span
						>
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
			{#if results.dnsRecords.txtRecords?.length > 0}
				<div class="overflow-x-auto w-full">
					<h3 class="text-gray-400 mb-2 mt-4">TXT Records</h3>
					<ul class="space-y-1">
						{#each results.dnsRecords.txtRecords as record}
							<li class="text-gray-300">• {record}</li>
						{/each}
					</ul>
				</div>
			{/if}
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
							<div
								class={results.robotsTxt.contentType === 'text/plain'
									? 'text-green-400'
									: 'text-red-400'}
							>
								{results.robotsTxt.contentType || 'N/A'}
							</div>
						</div>
					</div>

					{#if results.robotsTxt.exists && results.robotsTxt.contentType === 'text/plain'}
						<div>
							<h3 class="text-gray-400 mb-2">Content</h3>
							<pre
								class="p-3 bg-gray-800/50 rounded-lg overflow-x-auto text-sm whitespace-pre-wrap">{results
									.robotsTxt.content}</pre>
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
							<div
								class={results.securityTxt.contentType?.startsWith('text/plain')
									? 'text-green-400'
									: 'text-red-400'}
							>
								{results.securityTxt.contentType || 'N/A'}
							</div>
						</div>
						{#if results.securityTxt.exists}
							<div>
								<div class="text-gray-400 mb-1">PGP Signature</div>
								<div
									class={results.securityTxt.validSignature ? 'text-green-400' : 'text-yellow-400'}
								>
									{results.securityTxt.validSignature ? 'Signed' : 'Not Signed'}
								</div>
							</div>
						{/if}
					</div>

					{#if results.securityTxt.exists && results.securityTxt.contentType?.startsWith('text/plain')}
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
							{@const daysUntilExpiry = Math.floor(
								(new Date(results.securityTxt.expiration) - new Date()) / (1000 * 60 * 60 * 24)
							)}
							<div>
								<h3 class="text-gray-400 mb-2">Expiration</h3>
								<div class="p-2 bg-gray-800/50 rounded-lg">
									<div class="font-medium">
										{new Date(results.securityTxt.expiration).toLocaleDateString('no')}
									</div>
									<div
										class={daysUntilExpiry < 30
											? 'text-red-400'
											: daysUntilExpiry < 90
												? 'text-yellow-400'
												: 'text-green-400'}
									>
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
							<pre
								class="p-3 bg-gray-800/50 rounded-lg overflow-x-auto text-sm whitespace-pre-wrap">{results
									.securityTxt.content}</pre>
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
{/if}
<style>
  .w-autofill {
    width: -webkit-fill-available;
    width: inherit;
  }

	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.5;
		}
	}

	.animate-pulse {
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	.transition-opacity {
		transition-property: opacity;
		transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
	}

	.duration-300 {
		transition-duration: 300ms;
	}
</style>
