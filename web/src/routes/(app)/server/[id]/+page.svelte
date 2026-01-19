<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Server } from '$lib/models';
	import StatusIndicator from '$lib/components/server/StatusIndicator.svelte';
	import StatusMetrics from '$lib/components/server/StatusMetrics.svelte';
	import ResponseTimeGraph from '$lib/components/server/ResponseTimeGraph.svelte';
	import ServerInfoCard from '$lib/components/server/ServerInfoCard.svelte';
	import SecurityScan from '$lib/components/SecurityScan.svelte';
	import HealthProbes from '$lib/components/scan/HealthProbes.svelte';
	import ServerControls from '$lib/components/server/ServerControls.svelte';

	/** @type {import('./$types').PageData} */
	export let data;

	let serverID: number = 0;
	let server: Server | null = null;
	let isLoading = true;
	let isLoadingResults = true;
	let error: string | null = null;
	let searchResults = null;

	$: if (data.id) {
		serverID = data.id;
	}

	onMount(() => {
		fetchServerData(serverID);
		fetchServerReport(serverID);
	});

	async function fetchServerReport(id: number) {
		try {
			const response = await fetch(`/api/servers/${id}/report`);
			if (!response.ok) throw new Error('Failed to fetch server data');
			searchResults = await response.json();
		} finally {
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
	{#if isLoading}
		<div class="loading-state">
			<div class="loading-spinner"></div>
			<span>Loading server data...</span>
		</div>
	{:else if error}
		<div class="error-state">
			<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
			</svg>
			<span>Error: {error}</span>
		</div>
	{:else if server}
		{@const metrics = calculateMetrics(server, searchResults)}
		<!-- Hero Header -->
		<header class="dashboard-header">
			<div class="header-content">
				<div class="header-title">
					<div class="server-url-badge">
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
				<StatusIndicator pingResults={server.ping_results} />
			</div>
		</header>

		<!-- Main Dashboard Grid -->
		<div class="dashboard-grid">
			<!-- Metrics Row - Bento Style -->
			<section class="metrics-section">
				<div class="metric-card accent-blue">
					<div class="metric-icon">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M13 10V3L4 14h7v7l9-11h-7z" />
						</svg>
					</div>
					<div class="metric-content">
						<span class="metric-value">{metrics.currentResponse}<span class="metric-unit">ms</span></span>
						<span class="metric-label">Current Response</span>
					</div>
				</div>

				<div class="metric-card accent-purple">
					<div class="metric-icon">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
						</svg>
					</div>
					<div class="metric-content">
						<span class="metric-value">{metrics.avgResponse}<span class="metric-unit">ms</span></span>
						<span class="metric-label">24h Average</span>
					</div>
				</div>

				<div class="metric-card accent-green">
					<div class="metric-icon">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
					</div>
					<div class="metric-content">
						<span class="metric-value">{metrics.uptimeDay}<span class="metric-unit">%</span></span>
						<span class="metric-label">24h Uptime</span>
					</div>
				</div>

				<div class="metric-card accent-cyan">
					<div class="metric-icon">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
					</div>
					<div class="metric-content">
						<span class="metric-value">{metrics.uptimeMonth}<span class="metric-unit">%</span></span>
						<span class="metric-label">30d Uptime</span>
					</div>
				</div>

				<div class="metric-card accent-orange {metrics.certDaysLeft < 30 ? 'warning' : ''}">
					<div class="metric-icon">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
						</svg>
					</div>
					<div class="metric-content">
						<span class="metric-value">{metrics.certDaysLeft > 0 ? metrics.certDaysLeft : 'N/A'}<span class="metric-unit">{metrics.certDaysLeft > 0 ? 'd' : ''}</span></span>
						<span class="metric-label">Cert Expires</span>
					</div>
				</div>
			</section>

			<!-- Two Column Layout: Graph + Config -->
			<div class="content-grid">
				<!-- Response Time Graph -->
				<section class="graph-section">
					<div class="section-header">
						<h2>Response Time</h2>
						<span class="section-badge">Live</span>
					</div>
					<ResponseTimeGraph
						data={server.ping_results.map((ping) => ({
							timestamp: new Date(ping.timestamp),
							value: ping.response_time_ms
						}))}
					/>
				</section>

				<!-- Server Configuration -->
				<aside class="config-section">
					<ServerInfoCard {server} />
				</aside>
			</div>
		</div>
	{/if}

	<!-- Security Scan Section -->
	<section class="security-section">
		{#if isLoadingResults}
			<div class="loading-card">
				<div class="loading-content">
					<div class="loading-spinner"></div>
					<span>Running security analysis...</span>
				</div>
			</div>
		{:else}
			<SecurityScan {searchResults} />
		{/if}
	</section>
</div>

<style>
	.dashboard-container {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		padding: 1.5rem;
		min-height: 100vh;
		background: linear-gradient(180deg, #0a0a0a 0%, #111111 100%);
	}

	/* Loading & Error States */
	.loading-state,
	.error-state {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		padding: 3rem;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.error-state {
		color: #ef4444;
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid rgba(239, 68, 68, 0.2);
		border-radius: 0.75rem;
	}

	.loading-spinner {
		width: 1.25rem;
		height: 1.25rem;
		border: 2px solid #333;
		border-top-color: #4ade80;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Header */
	.dashboard-header {
		background: linear-gradient(135deg, #1a1a1a 0%, #0d0d0d 100%);
		border: 1px solid #222;
		border-radius: 1rem;
		padding: 1.5rem;
		position: relative;
		overflow: hidden;
	}

	.dashboard-header::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 1px;
		background: linear-gradient(90deg, transparent, rgba(74, 222, 128, 0.3), transparent);
	}

	.header-content {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.header-title {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.server-url-badge {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
		font-size: 1.25rem;
		font-weight: 500;
	}

	.url-protocol {
		color: #6b7280;
	}

	.url-domain {
		color: #f9fafb;
		background: linear-gradient(90deg, #4ade80, #22d3ee);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}

	/* Metrics Section */
	.metrics-section {
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 1rem;
	}

	@media (max-width: 1200px) {
		.metrics-section {
			grid-template-columns: repeat(3, 1fr);
		}
	}

	@media (max-width: 768px) {
		.metrics-section {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	.metric-card {
		background: #141414;
		border: 1px solid #222;
		border-radius: 0.875rem;
		padding: 1.25rem;
		display: flex;
		align-items: flex-start;
		gap: 1rem;
		position: relative;
		overflow: hidden;
		transition: all 0.2s ease;
	}

	.metric-card:hover {
		border-color: #333;
		transform: translateY(-2px);
	}

	.metric-card::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 2px;
		background: var(--accent-color);
		opacity: 0.6;
	}

	.metric-card.accent-blue { --accent-color: #3b82f6; }
	.metric-card.accent-purple { --accent-color: #8b5cf6; }
	.metric-card.accent-green { --accent-color: #22c55e; }
	.metric-card.accent-cyan { --accent-color: #06b6d4; }
	.metric-card.accent-orange { --accent-color: #f59e0b; }
	.metric-card.warning { --accent-color: #ef4444; }

	.metric-icon {
		width: 2.5rem;
		height: 2.5rem;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(255, 255, 255, 0.03);
		border-radius: 0.625rem;
		flex-shrink: 0;
	}

	.metric-icon svg {
		width: 1.25rem;
		height: 1.25rem;
		stroke: var(--accent-color);
	}

	.metric-content {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.metric-value {
		font-size: 1.5rem;
		font-weight: 600;
		color: #f9fafb;
		line-height: 1;
		font-variant-numeric: tabular-nums;
	}

	.metric-unit {
		font-size: 0.875rem;
		font-weight: 400;
		color: #6b7280;
		margin-left: 0.125rem;
	}

	.metric-label {
		font-size: 0.75rem;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	/* Content Grid */
	.content-grid {
		display: grid;
		grid-template-columns: 1fr 380px;
		gap: 1.5rem;
	}

	@media (max-width: 1024px) {
		.content-grid {
			grid-template-columns: 1fr;
		}
	}

	/* Graph Section */
	.graph-section {
		background: #141414;
		border: 1px solid #222;
		border-radius: 1rem;
		overflow: hidden;
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.25rem;
		border-bottom: 1px solid #222;
	}

	.section-header h2 {
		font-size: 0.875rem;
		font-weight: 500;
		color: #d1d5db;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.section-badge {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.75rem;
		color: #4ade80;
		background: rgba(74, 222, 128, 0.1);
		padding: 0.25rem 0.625rem;
		border-radius: 9999px;
	}

	.section-badge::before {
		content: '';
		width: 0.375rem;
		height: 0.375rem;
		background: #4ade80;
		border-radius: 50%;
		animation: pulse-dot 2s ease-in-out infinite;
	}

	@keyframes pulse-dot {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}

	/* Config Section */
	.config-section {
		display: flex;
		flex-direction: column;
	}

	/* Security Section */
	.security-section {
		margin-top: 0.5rem;
	}

	.loading-card {
		background: #141414;
		border: 1px solid #222;
		border-radius: 1rem;
		padding: 3rem;
	}

	.loading-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1rem;
		color: #6b7280;
	}

	/* Dashboard Grid Container */
	.dashboard-grid {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}
</style>
