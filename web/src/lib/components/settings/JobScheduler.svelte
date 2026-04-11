<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Play, Clock, CheckCircle, XCircle, Loader, RefreshCw } from 'lucide-svelte';

	interface JobInfo {
		name: string;
		description: string;
		status: 'idle' | 'running' | 'success' | 'failed';
		last_run: string;
		last_duration_seconds: number;
		last_error: string;
		next_run: string;
		progress: string;
		schedule: string;
	}

	interface JobLog {
		id: number;
		job_name: string;
		trigger: string;
		status: string;
		started_at: string;
		ended_at: string;
		duration_seconds: number;
		summary: string;
		error: string;
	}

	let jobs: JobInfo[] = $state([]);
	let loading = $state(true);
	let expandedJob: string | null = $state(null);
	let jobLogs: JobLog[] = $state([]);
	let logsLoading = $state(false);
	let pollTimer: ReturnType<typeof setInterval>;

	onMount(() => {
		fetchJobs();
		pollTimer = setInterval(fetchJobs, 5000);
	});

	onDestroy(() => {
		clearInterval(pollTimer);
	});

	async function fetchJobs() {
		try {
			const res = await fetch('/api/jobs', { credentials: 'include' });
			if (res.ok) {
				jobs = await res.json();
			}
		} catch {
			// silent
		} finally {
			loading = false;
		}
	}

	async function triggerJob(name: string) {
		// Optimistic: immediately show running state
		jobs = jobs.map((j) =>
			j.name === name ? { ...j, status: 'running' as const, progress: 'starting...' } : j
		);
		await fetch(`/api/jobs/${name}/trigger`, {
			method: 'POST',
			credentials: 'include'
		});
		// Poll faster while running
		pollWhileRunning(name);
	}

	function pollWhileRunning(name: string) {
		const fast = setInterval(async () => {
			await fetchJobs();
			const job = jobs.find((j) => j.name === name);
			if (!job || job.status !== 'running') {
				clearInterval(fast);
				// Refresh logs if expanded
				if (expandedJob === name) {
					toggleLogs(name);
					toggleLogs(name);
				}
			}
		}, 500);
		// Safety: stop after 5 minutes
		setTimeout(() => clearInterval(fast), 5 * 60 * 1000);
	}

	async function toggleLogs(name: string) {
		if (expandedJob === name) {
			expandedJob = null;
			return;
		}
		expandedJob = name;
		logsLoading = true;
		try {
			const res = await fetch(`/api/jobs/${name}/logs?limit=10`, { credentials: 'include' });
			if (res.ok) {
				jobLogs = await res.json();
			}
		} catch {
			jobLogs = [];
		} finally {
			logsLoading = false;
		}
	}

	function formatDuration(seconds: number): string {
		if (!seconds) return '—';
		if (seconds < 1) return `${Math.round(seconds * 1000)}ms`;
		if (seconds < 60) return `${seconds.toFixed(1)}s`;
		const m = Math.floor(seconds / 60);
		const s = Math.round(seconds % 60);
		return `${m}m ${s}s`;
	}

	function formatTime(iso: string): string {
		if (!iso || iso === '0001-01-01T00:00:00Z') return '—';
		return new Date(iso).toLocaleString();
	}

	function timeAgo(iso: string): string {
		if (!iso || iso === '0001-01-01T00:00:00Z') return 'never';
		const diff = Date.now() - new Date(iso).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `${hours}h ago`;
		return `${Math.floor(hours / 24)}d ago`;
	}

	function timeUntil(iso: string): string {
		if (!iso || iso === '0001-01-01T00:00:00Z') return '—';
		const diff = new Date(iso).getTime() - Date.now();
		if (diff < 0) return 'due';
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return '<1m';
		if (mins < 60) return `${mins}m`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `${hours}h ${mins % 60}m`;
		return `${Math.floor(hours / 24)}d`;
	}
</script>

<div class="job-scheduler">
	<div class="header">
		<h3>Scheduled Jobs</h3>
		<button class="refresh-btn" onclick={fetchJobs} title="Refresh">
			<RefreshCw size={14} />
		</button>
	</div>

	{#if loading}
		<div class="loading">
			<Loader size={20} class="spin" />
			<span>Loading jobs...</span>
		</div>
	{:else if jobs.length === 0}
		<p class="empty">No jobs registered.</p>
	{:else}
		<div class="job-list">
			{#each jobs as job}
				<div class="job-card" class:running={job.status === 'running'}>
					<div class="job-main">
						<div class="job-left">
							<div class="job-status-icon">
								{#if job.status === 'running'}
									<Loader size={16} class="spin" />
								{:else if job.status === 'success'}
									<CheckCircle size={16} />
								{:else if job.status === 'failed'}
									<XCircle size={16} />
								{:else}
									<Clock size={16} />
								{/if}
							</div>
							<div class="job-info">
								<button class="job-name" onclick={() => toggleLogs(job.name)}>
									{job.name}
								</button>
								<span class="job-desc">{job.description}</span>
							</div>
						</div>
						<div class="job-right">
							<div class="job-meta">
								<span class="job-schedule">{job.schedule}</span>
								<span class="job-timing">
									{#if job.status === 'running' && job.progress}
										<span class="progress-text">{job.progress}</span>
									{:else}
										last: {timeAgo(job.last_run)} · next: {timeUntil(job.next_run)}
									{/if}
								</span>
								{#if job.last_error}
									<span class="job-error">{job.last_error}</span>
								{/if}
							</div>
							<div class="job-actions">
								{#if job.status === 'running'}
									<div class="action-btn running-indicator" title="Running...">
										<Loader size={14} class="spin" />
									</div>
								{:else}
									<button class="action-btn trigger" onclick={() => triggerJob(job.name)} title="Run now">
										<Play size={14} />
									</button>
								{/if}
							</div>
						</div>
					</div>

					{#if expandedJob === job.name}
						<div class="job-logs">
							{#if logsLoading}
								<div class="loading-inline"><Loader size={14} class="spin" /> Loading...</div>
							{:else if jobLogs.length === 0}
								<p class="empty-logs">No run history yet.</p>
							{:else}
								<table class="logs-table">
									<thead>
										<tr>
											<th>Time</th>
											<th>Trigger</th>
											<th>Status</th>
											<th>Duration</th>
											<th>Summary</th>
										</tr>
									</thead>
									<tbody>
										{#each jobLogs as log}
											<tr class:log-failed={log.status === 'failed'}>
												<td class="log-time">{formatTime(log.started_at)}</td>
												<td><span class="trigger-badge {log.trigger}">{log.trigger}</span></td>
												<td><span class="status-badge {log.status}">{log.status}</span></td>
												<td class="log-duration">{formatDuration(log.duration_seconds)}</td>
												<td class="log-summary">{log.error || log.summary || '—'}</td>
											</tr>
										{/each}
									</tbody>
								</table>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.job-scheduler {
		padding: 1rem 1.5rem;
	}

	.header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.header h3 {
		font-size: 1rem;
		font-weight: 600;
		color: #e5e7eb;
	}

	.refresh-btn {
		display: flex;
		align-items: center;
		padding: 0.375rem;
		border-radius: 0.375rem;
		color: #6b7280;
		background: none;
		border: none;
		cursor: pointer;
		transition: color 0.15s;
	}
	.refresh-btn:hover { color: #e5e7eb; }

	.loading, .empty {
		text-align: center;
		color: #6b7280;
		padding: 2rem 0;
		font-size: 0.875rem;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
	}

	.job-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.job-card {
		background: #1a1a1a;
		border: 1px solid #2a2a2a;
		border-radius: 0.5rem;
		overflow: hidden;
		transition: border-color 0.15s;
	}
	.job-card:hover { border-color: #333; }
	.job-card.running { border-color: rgba(59, 130, 246, 0.3); }

	.job-main {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		gap: 1rem;
	}

	.job-left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		min-width: 0;
	}

	.job-status-icon {
		flex-shrink: 0;
		display: flex;
	}
	.job-card:has(.job-status-icon) :global(.spin) {
		animation: spin 1s linear infinite;
	}
	:global(.spin) {
		animation: spin 1s linear infinite;
	}
	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.job-card .job-status-icon { color: #6b7280; }
	.job-card.running .job-status-icon { color: #3b82f6; }
	.job-card:has(.status-badge) .job-status-icon { color: #6b7280; }

	/* Status icon colors based on job status */
	.job-card :global(.lucide-check-circle) { color: #22c55e; }
	.job-card :global(.lucide-x-circle) { color: #ef4444; }
	.job-card :global(.lucide-clock) { color: #6b7280; }
	.job-card :global(.lucide-loader) { color: #3b82f6; }

	.job-info {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.job-name {
		font-size: 0.8125rem;
		font-weight: 500;
		color: #e5e7eb;
		background: none;
		border: none;
		padding: 0;
		cursor: pointer;
		text-align: left;
		font-family: ui-monospace, monospace;
	}
	.job-name:hover { color: #3b82f6; }

	.job-desc {
		font-size: 0.6875rem;
		color: #6b7280;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.job-right {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex-shrink: 0;
	}

	.job-meta {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 2px;
	}

	.job-schedule {
		font-size: 0.6875rem;
		color: #9ca3af;
		font-weight: 500;
	}

	.job-timing {
		font-size: 0.625rem;
		color: #4b5563;
	}

	.progress-text {
		color: #3b82f6;
	}

	.job-error {
		font-size: 0.625rem;
		color: #ef4444;
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.job-actions {
		display: flex;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 0.375rem;
		border: 1px solid #333;
		background: #252525;
		cursor: pointer;
		transition: all 0.15s;
	}
	.action-btn.trigger { color: #22c55e; }
	.action-btn.trigger:hover { background: rgba(34, 197, 94, 0.15); border-color: rgba(34, 197, 94, 0.3); }
	.action-btn.running-indicator { color: #3b82f6; border-color: rgba(59, 130, 246, 0.3); background: rgba(59, 130, 246, 0.1); cursor: default; }

	/* Logs */
	.job-logs {
		border-top: 1px solid #2a2a2a;
		padding: 0.75rem 1rem;
		background: #161616;
	}

	.loading-inline {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #6b7280;
		font-size: 0.75rem;
		padding: 0.5rem 0;
	}

	.empty-logs {
		color: #4b5563;
		font-size: 0.75rem;
		text-align: center;
		padding: 0.5rem 0;
	}

	.logs-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.6875rem;
	}

	.logs-table th {
		text-align: left;
		color: #6b7280;
		font-weight: 500;
		padding: 0.25rem 0.5rem;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		border-bottom: 1px solid #2a2a2a;
	}

	.logs-table td {
		padding: 0.375rem 0.5rem;
		color: #9ca3af;
		border-bottom: 1px solid #1f1f1f;
	}

	.log-time { white-space: nowrap; }
	.log-duration { font-variant-numeric: tabular-nums; }
	.log-summary {
		max-width: 300px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	tr.log-failed td { color: #f87171; }

	.status-badge, .trigger-badge {
		display: inline-block;
		padding: 0.125rem 0.375rem;
		border-radius: 9999px;
		font-size: 0.625rem;
		font-weight: 500;
	}
	.status-badge.success { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
	.status-badge.failed { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
	.status-badge.running { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
	.status-badge.idle { background: rgba(107, 114, 128, 0.15); color: #6b7280; }
	.trigger-badge.manual { background: rgba(168, 85, 247, 0.15); color: #a855f7; }
	.trigger-badge.scheduled { background: rgba(107, 114, 128, 0.15); color: #6b7280; }
</style>
