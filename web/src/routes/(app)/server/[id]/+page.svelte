<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Server } from '$lib/models';
	import StatusIndicator from '$lib/components/server/StatusIndicator.svelte';
	import SecurityScan from '$lib/components/SecurityScan.svelte';
	import ServerControls from '$lib/components/server/ServerControls.svelte';

	/** @type {import('./$types').PageData} */
	export let data;

	let serverID: number = 0;
	let server: Server | null = null;
	let isLoading = true;
	let isLoadingResults = true;
	let error: string | null = null;
	let searchResults = null;
	let reportStatus: { state: string; startedAt: string; completedAt: string } | null = null;
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	$: if (data.id) {
		serverID = data.id;
	}

	onMount(() => {
		fetchServerData(serverID);
		fetchServerReport(serverID);
	});

	onDestroy(() => {
		if (pollInterval) {
			clearInterval(pollInterval);
		}
	});

	async function fetchServerReport(id: number) {
		try {
			const response = await fetch(`/api/servers/${id}/report`);
			if (!response.ok) throw new Error('Failed to fetch report');

			const data = await response.json();
			reportStatus = data.status;

			if (data.status?.state === 'done') {
				searchResults = data.report;
				isLoadingResults = false;
				if (pollInterval) {
					clearInterval(pollInterval);
					pollInterval = null;
				}
			} else {
				// Poll every 2 seconds while report is generating
				if (!pollInterval) {
					pollInterval = setInterval(() => fetchServerReport(id), 2000);
				}
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to fetch report';
			isLoadingResults = false;
		}
	}

	async function fetchServerData(id: number) {
		isLoading = true;
		error = null;

		try {
			const response = await fetch(`/api/servers/${id}`);
			if (!response.ok) throw new Error('Failed to fetch server data');

			const data: Server = await response.json();
			server = data;
		} catch (e) {
			error = e instanceof Error ? e.message : 'An error occurred';
			server = null;
		} finally {
			isLoading = false;
		}
	}

	// Server management functions
	async function handleServerUpdate(event: CustomEvent) {
		const { data: updatedServer } = event.detail;
		isLoading = true;

		try {
			const response = await fetch(`/api/servers/${serverID}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(updatedServer)
			});

			if (!response.ok) throw new Error('Failed to update server');

			// Update server state locally instead of reloading
			if (server) {
				Object.assign(server, updatedServer);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update server';
		} finally {
			isLoading = false;
		}
	}

	async function handleToggleActive(event: CustomEvent) {
		const { active } = event.detail;
		isLoading = true;

		try {
			const response = await fetch(`/api/servers/${serverID}`, {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ active })
			});

			if (!response.ok) throw new Error('Failed to update server status');

			// Update server state locally instead of reloading
			if (server) {
				server.active = active;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update server status';
		} finally {
			isLoading = false;
		}
	}

	async function handleDelete() {
		isLoading = true;

		try {
			const response = await fetch(`/api/servers/${serverID}`, {
				method: 'DELETE'
			});

			if (!response.ok) throw new Error('Failed to delete server');

			// Navigate back to servers list
			goto('/dashboard');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete server';
			isLoading = false;
		}
	}

	function getLatestPing(server: Server) {
		return server.ping_results[0] || null;
	}

	function isPingSuccessful(ping: { status_code: number }) {
		return ping.status_code > 0 && ping.status_code < 400;
	}

	function getGradeColor(grade: string): string {
		switch (grade?.toUpperCase()) {
			case 'A+':
			case 'A':
				return 'grade-a';
			case 'B':
				return 'grade-b';
			case 'C':
				return 'grade-c';
			case 'D':
				return 'grade-d';
			case 'E':
			case 'F':
				return 'grade-f';
			default:
				return 'grade-unknown';
		}
	}

	function getRiskColor(risk: string): string {
		switch (risk?.toLowerCase()) {
			case 'info':
			case 'low':
				return 'risk-low';
			case 'medium':
				return 'risk-medium';
			case 'high':
			case 'critical':
				return 'risk-high';
			default:
				return 'risk-unknown';
		}
	}

	function calculateMetrics(server: Server, reportData: typeof searchResults) {
		const decimals = 3;
		const latestPing = getLatestPing(server);
		const last24hPings = server.ping_results.filter(
			(ping) => new Date(ping.timestamp).getTime() > Date.now() - 24 * 60 * 60 * 1000
		);

		const avgResponse = Math.round(
			last24hPings.reduce((acc, ping) => acc + ping.response_time_ms, 0) / last24hPings.length
		);

		const uptimeDay = Number(
			(
				(last24hPings.filter((ping) => isPingSuccessful(ping)).length / last24hPings.length) *
				100
			).toFixed(decimals)
		);

		const last30DayPings = server.ping_results.filter(
			(ping) => new Date(ping.timestamp).getTime() > Date.now() - 30 * 24 * 60 * 60 * 1000
		);

		const uptimeMonth = Number(
			(
				(last30DayPings.filter((ping) => isPingSuccessful(ping)).length / last30DayPings.length) *
				100
			).toFixed(decimals)
		);

		const certValidUntil = reportData?.certificate?.validUntil;

		return {
			currentResponse: latestPing?.response_time_ms || 0,
			avgResponse,
			uptimeDay,
			uptimeMonth,
			certDaysLeft: certValidUntil
				? Math.ceil((new Date(certValidUntil).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
				: 0,
			certExpDate: certValidUntil
		};
	}
</script>

<div class="dashboard-container">
	{#if isLoading || isLoadingResults}
		<div class="loading-state">
			<div class="loading-spinner"></div>
			{#if reportStatus?.state === 'running'}
				<span>Generating security report...</span>
			{:else}
				<span>Loading security report...</span>
			{/if}
		</div>
	{:else if error}
		<div class="error-state">
			<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
			</svg>
			<span>Error: {error}</span>
		</div>
	{:else if server && searchResults}
		{@const metrics = calculateMetrics(server, searchResults)}

		<!-- Hero Section: Screenshot + Security Overview -->
		<section class="hero-section">
			<!-- Screenshot (2/3) -->
			<div class="hero-screenshot">
				<img
					src={`/api/screenshot/${encodeURIComponent(server.url)}`}
					alt="Website preview"
					class="screenshot-img"
					on:error={(e) => e.target.style.display = 'none'}
				/>
				<div class="screenshot-overlay"></div>
			</div>

			<!-- Security Info (1/3) -->
			<div class="hero-info">
				<div class="hero-header">
					<div class="server-url">
						<span class="url-protocol">https://</span>
						<span class="url-domain">{server.url}</span>
					</div>
					<ServerControls
						{server}
						{isLoading}
						on:update={handleServerUpdate}
						on:toggleActive={handleToggleActive}
						on:delete={handleDelete}
					/>
				</div>

				<!-- Security Grades -->
				<div class="security-grades">
					<div class="grade-card">
						<span class="grade-value {getGradeColor(searchResults.headers?.score)}">{searchResults.headers?.score || '?'}</span>
						<span class="grade-label">Headers</span>
					</div>
					<div class="grade-card">
						<span class="grade-value {getGradeColor(searchResults.certificate?.grade)}">{searchResults.certificate?.grade || '?'}</span>
						<span class="grade-label">SSL/TLS</span>
					</div>
					<div class="grade-card">
						<span class="grade-value {getRiskColor(searchResults.adminPages?.risk)}">{searchResults.adminPages?.risk || '?'}</span>
						<span class="grade-label">Admin</span>
					</div>
					<div class="grade-card">
						<span class="grade-value {getRiskColor(searchResults.swagger?.risk)}">{searchResults.swagger?.risk || '?'}</span>
						<span class="grade-label">API</span>
					</div>
					<div class="grade-card">
						<span class="grade-value {getRiskColor(searchResults.secretExposure?.risk)}">{searchResults.secretExposure?.risk || '?'}</span>
						<span class="grade-label">Secrets</span>
					</div>
				</div>

				<!-- Key Findings -->
				<div class="key-findings">
					<h3>Key Findings</h3>
					<ul>
						{#if searchResults.headers?.issues?.length > 0}
							<li class="finding-item warning">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
								{searchResults.headers.issues.length} header issue{searchResults.headers.issues.length > 1 ? 's' : ''} detected
							</li>
						{/if}
						<li class="finding-item {metrics.certDaysLeft < 30 ? 'warning' : 'success'}">
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>
							Certificate expires in {metrics.certDaysLeft} days
						</li>
						{#if searchResults.adminPages?.exposed?.length > 0}
							<li class="finding-item danger">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" /></svg>
								{searchResults.adminPages.exposed.length} admin page{searchResults.adminPages.exposed.length > 1 ? 's' : ''} exposed
							</li>
						{:else}
							<li class="finding-item success">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
								No admin pages exposed
							</li>
						{/if}
						{#if searchResults.swagger?.exposed}
							<li class="finding-item warning">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
								API documentation publicly accessible
							</li>
						{:else}
							<li class="finding-item success">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
								No public API documentation
							</li>
						{/if}
						{#if searchResults.secretExposure?.findings?.length > 0}
							<li class="finding-item danger">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" /></svg>
								{searchResults.secretExposure.findings.length} secret{searchResults.secretExposure.findings.length > 1 ? 's' : ''} exposed
							</li>
						{:else}
							<li class="finding-item success">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
								No secrets exposed
							</li>
						{/if}
					</ul>
				</div>

				<!-- Quick Stats -->
				<div class="quick-stats">
					<div class="stat">
						<span class="stat-value">{metrics.uptimeMonth}%</span>
						<span class="stat-label">Uptime</span>
					</div>
					<div class="stat">
						<span class="stat-value">{metrics.avgResponse}ms</span>
						<span class="stat-label">Response</span>
					</div>
					<div class="stat">
						<span class="stat-value">{server.ping_results?.length || 0}</span>
						<span class="stat-label">Checks</span>
					</div>
				</div>
			</div>
		</section>

		<!-- Uptime Bar -->
		<section class="uptime-section">
			<StatusIndicator pingResults={server.ping_results} />
		</section>

		<!-- Security Details -->
		<section class="security-details">
			<SecurityScan {searchResults} hideHero={true} />
		</section>
	{/if}
</div>

<style>
	.dashboard-container {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		padding: 1.5rem;
		min-height: 100vh;
		width: 100%;
	}

	/* Loading & Error States */
	.loading-state,
	.error-state {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		padding: 4rem;
		color: #9ca3af;
		font-size: 0.875rem;
	}

	.error-state {
		color: #ef4444;
		background: #202020;
		border-radius: 0.5rem;
	}

	.loading-spinner {
		width: 1.5rem;
		height: 1.5rem;
		border: 2px solid #2b2b2b;
		border-top-color: #22c55e;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Hero Section */
	.hero-section {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 1rem;
		min-height: 400px;
	}

	@media (max-width: 1024px) {
		.hero-section {
			grid-template-columns: 1fr;
			min-height: auto;
		}
	}

	.hero-screenshot {
		position: relative;
		background: #202020;
		border-radius: 0.5rem;
		overflow: hidden;
	}

	.screenshot-img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		object-position: top;
	}

	.screenshot-overlay {
		position: absolute;
		inset: 0;
		background: linear-gradient(to top, rgba(32, 32, 32, 0.8) 0%, transparent 30%);
		pointer-events: none;
	}

	/* Hero Info Panel */
	.hero-info {
		background: #202020;
		border-radius: 0.5rem;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.hero-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1rem;
	}

	.server-url {
		font-family: ui-monospace, monospace;
		font-size: 0.875rem;
		word-break: break-all;
	}

	.url-protocol {
		color: #6b7280;
	}

	.url-domain {
		color: #e5e7eb;
	}

	/* Security Grades */
	.security-grades {
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 0.5rem;
	}

	.grade-card {
		background: #2b2b2b;
		border-radius: 0.5rem;
		padding: 0.75rem 0.5rem;
		text-align: center;
	}

	.grade-value {
		display: block;
		font-size: 1.5rem;
		font-weight: 700;
		text-transform: uppercase;
	}

	.grade-label {
		display: block;
		font-size: 0.625rem;
		color: #9ca3af;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin-top: 0.25rem;
	}

	/* Grade Colors */
	.grade-a { color: #22c55e; }
	.grade-b { color: #84cc16; }
	.grade-c { color: #eab308; }
	.grade-d { color: #f97316; }
	.grade-f { color: #ef4444; }
	.grade-unknown { color: #6b7280; }

	/* Risk Colors */
	.risk-low { color: #22c55e; }
	.risk-medium { color: #eab308; }
	.risk-high { color: #ef4444; }
	.risk-unknown { color: #6b7280; }

	/* Key Findings */
	.key-findings {
		flex: 1;
	}

	.key-findings h3 {
		font-size: 0.6875rem;
		font-weight: 500;
		color: #9ca3af;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin-bottom: 0.75rem;
	}

	.key-findings ul {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.finding-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		color: #d1d5db;
	}

	.finding-item svg {
		width: 1rem;
		height: 1rem;
		flex-shrink: 0;
	}

	.finding-item.success svg { stroke: #22c55e; }
	.finding-item.warning svg { stroke: #eab308; }
	.finding-item.danger svg { stroke: #ef4444; }

	/* Quick Stats */
	.quick-stats {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.5rem;
		padding-top: 1rem;
		border-top: 1px solid #2b2b2b;
	}

	.stat {
		text-align: center;
	}

	.stat-value {
		display: block;
		font-size: 1rem;
		font-weight: 600;
		color: #e5e7eb;
		font-variant-numeric: tabular-nums;
	}

	.stat-label {
		display: block;
		font-size: 0.625rem;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		margin-top: 0.125rem;
	}

	/* Uptime Section */
	.uptime-section {
		background: #202020;
		border-radius: 0.5rem;
		padding: 0.75rem 1rem;
	}

	/* Security Details */
	.security-details {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
</style>
