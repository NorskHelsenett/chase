<script lang="ts">
	import type { Server } from '$lib/models';
	import { formatDistanceToNow } from 'date-fns';
	import { Globe, ExternalLink } from 'lucide-svelte';

	export let server: Server;

	$: nextCheckIn = formatDistanceToNow(new Date(server.next_check), { addSuffix: true });

	function getLatestPingResult(server) {
		if (!server.ping_results?.length) return null;

		return server.ping_results.reduce((latest, current) => {
			const latestTime = new Date(latest.timestamp).getTime();
			const currentTime = new Date(current.timestamp).getTime();
			return currentTime > latestTime ? current : latest;
		}, server.ping_results[0]);
	}

	function getLatestStatusCode(server) {
		const latestPing = getLatestPingResult(server);
		return latestPing?.status_code ?? 'N/A';
	}

	function getStatusColor(server) {
		const latestPing = getLatestPingResult(server);
		if (!latestPing) return 'neutral';
		return latestPing.status_code === server.expected_status ? 'success' : 'error';
	}
</script>

<div class="config-card">
	<div class="config-header">
		<h2>Configuration</h2>
		<a
			href={`https://${server.url}`}
			target="_blank"
			rel="noopener noreferrer"
			class="external-link"
		>
			<ExternalLink size={14} />
		</a>
	</div>

	<div class="config-grid">
		<!-- Status Row -->
		<div class="config-item full-width">
			<div class="config-label">
				<Globe size={14} />
				<span>Endpoint URL</span>
			</div>
			<div class="config-value url">
				<span class="url-text">{server.url}</span>
			</div>
		</div>

		<!-- Two column items -->
		<div class="config-item">
			<div class="config-label">Monitoring</div>
			<div class="config-value">
				<span class="status-badge {server.active ? 'active' : 'inactive'}">
					{server.active ? 'Active' : 'Paused'}
				</span>
			</div>
		</div>

		<div class="config-item">
			<div class="config-label">Next Check</div>
			<div class="config-value mono">{server.active ? nextCheckIn : '—'}</div>
		</div>

		<div class="config-item">
			<div class="config-label">TLS Verify</div>
			<div class="config-value">
				<span class="status-badge {server.allow_insecure ? 'warning' : 'success'}">
					{server.allow_insecure ? 'Disabled' : 'Enforced'}
				</span>
			</div>
		</div>

		<div class="config-item">
			<div class="config-label">Redirects</div>
			<div class="config-value">
				<span class="status-badge {server.follow_redirect ? 'success' : 'neutral'}">
					{server.follow_redirect ? 'Follow' : 'Block'}
				</span>
			</div>
		</div>

		<div class="config-item">
			<div class="config-label">HTTP Status</div>
			<div class="config-value">
				<span class="status-code {getStatusColor(server)}">
					{getLatestStatusCode(server)}
				</span>
			</div>
		</div>

		<div class="config-item">
			<div class="config-label">Expected</div>
			<div class="config-value mono">{server.expected_status}</div>
		</div>

		<div class="config-item">
			<div class="config-label">First Seen</div>
			<div class="config-value mono">{new Date(server.CreatedAt).toLocaleDateString()}</div>
		</div>

		<div class="config-item">
			<div class="config-label">Interval</div>
			<div class="config-value mono">{server.update_interval}s</div>
		</div>
	</div>

	{#if server.comment}
		<div class="config-comment">
			<div class="config-label">Notes</div>
			<p>{server.comment}</p>
		</div>
	{/if}
</div>

<style>
	.config-card {
		background: #141414;
		border: 1px solid #222;
		border-radius: 1rem;
		overflow: hidden;
		height: 100%;
		display: flex;
		flex-direction: column;
	}

	.config-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.25rem;
		border-bottom: 1px solid #222;
	}

	.config-header h2 {
		font-size: 0.875rem;
		font-weight: 500;
		color: #d1d5db;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.external-link {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.75rem;
		height: 1.75rem;
		border-radius: 0.5rem;
		background: rgba(255, 255, 255, 0.03);
		color: #6b7280;
		transition: all 0.15s ease;
	}

	.external-link:hover {
		background: rgba(255, 255, 255, 0.08);
		color: #4ade80;
	}

	.config-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1px;
		background: #222;
		flex: 1;
	}

	.config-item {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 1rem;
		background: #141414;
	}

	.config-item.full-width {
		grid-column: span 2;
	}

	.config-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.6875rem;
		font-weight: 500;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.config-value {
		font-size: 0.875rem;
		color: #e5e7eb;
	}

	.config-value.mono {
		font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
		font-size: 0.8125rem;
	}

	.config-value.url {
		font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
		font-size: 0.8125rem;
	}

	.url-text {
		color: #60a5fa;
		word-break: break-all;
	}

	.status-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.25rem 0.625rem;
		border-radius: 9999px;
		font-size: 0.6875rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.025em;
	}

	.status-badge.active {
		background: rgba(74, 222, 128, 0.15);
		color: #4ade80;
	}

	.status-badge.inactive {
		background: rgba(107, 114, 128, 0.15);
		color: #9ca3af;
	}

	.status-badge.success {
		background: rgba(74, 222, 128, 0.15);
		color: #4ade80;
	}

	.status-badge.warning {
		background: rgba(251, 191, 36, 0.15);
		color: #fbbf24;
	}

	.status-badge.neutral {
		background: rgba(107, 114, 128, 0.15);
		color: #9ca3af;
	}

	.status-code {
		font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
		font-size: 0.875rem;
		font-weight: 600;
	}

	.status-code.success {
		color: #4ade80;
	}

	.status-code.error {
		color: #f87171;
	}

	.status-code.neutral {
		color: #6b7280;
	}

	.config-comment {
		padding: 1rem;
		border-top: 1px solid #222;
		background: rgba(255, 255, 255, 0.02);
	}

	.config-comment p {
		font-size: 0.8125rem;
		color: #9ca3af;
		line-height: 1.5;
		margin-top: 0.5rem;
	}
</style>
