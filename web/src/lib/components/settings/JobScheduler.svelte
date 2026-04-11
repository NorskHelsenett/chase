<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		Play,
		Clock,
		CheckCircle,
		XCircle,
		Loader,
		Server,
		ServerOff,
		Users,
		Database,
		Activity,
		Zap
	} from 'lucide-svelte';

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

	interface SystemStats {
		active_servers: number;
		inactive_servers: number;
		total_pings: number;
		users: number;
		database_bytes: number;
		total_jobs: number;
		running_jobs: number;
	}

	let jobs: JobInfo[] = $state([]);
	let stats: SystemStats | null = $state(null);
	let loading = $state(true);
	let expandedJob: string | null = $state(null);
	let jobLogs: JobLog[] = $state([]);
	let logsLoading = $state(false);
	let pollTimer: ReturnType<typeof setInterval>;

	onMount(() => {
		fetchAll();
		pollTimer = setInterval(fetchJobs, 5000);
	});

	onDestroy(() => {
		clearInterval(pollTimer);
	});

	async function fetchAll() {
		await Promise.all([fetchJobs(), fetchStats()]);
		loading = false;
	}

	async function fetchJobs() {
		try {
			const res = await fetch('/api/jobs', { credentials: 'include' });
			if (res.ok) jobs = await res.json();
		} catch {
			/* silent */
		}
	}

	async function fetchStats() {
		try {
			const res = await fetch('/api/system-stats', { credentials: 'include' });
			if (res.ok) stats = await res.json();
		} catch {
			/* silent */
		}
	}

	async function triggerJob(name: string) {
		jobs = jobs.map((j) =>
			j.name === name ? { ...j, status: 'running' as const, progress: 'starting...' } : j
		);
		await fetch(`/api/jobs/${name}/trigger`, { method: 'POST', credentials: 'include' });
		pollWhileRunning(name);
	}

	function pollWhileRunning(name: string) {
		const fast = setInterval(async () => {
			await fetchJobs();
			const job = jobs.find((j) => j.name === name);
			if (!job || job.status !== 'running') {
				clearInterval(fast);
				if (expandedJob === name) {
					expandedJob = null;
					await loadLogs(name);
				}
				fetchStats();
			}
		}, 500);
		setTimeout(() => clearInterval(fast), 5 * 60 * 1000);
	}

	async function loadLogs(name: string) {
		expandedJob = name;
		logsLoading = true;
		try {
			const res = await fetch(`/api/jobs/${name}/logs?limit=10`, { credentials: 'include' });
			if (res.ok) jobLogs = await res.json();
		} catch {
			jobLogs = [];
		} finally {
			logsLoading = false;
		}
	}

	function toggleLogs(name: string) {
		if (expandedJob === name) {
			expandedJob = null;
		} else {
			loadLogs(name);
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

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${(bytes / Math.pow(k, i)).toFixed(i > 1 ? 1 : 0)} ${sizes[i]}`;
	}

	function formatNumber(n: number): string {
		if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
		if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
		return String(n);
	}

	function parseProgress(progress: string): { current: number; total: number } | null {
		const match = progress.match(/^(\d+)\s*\/\s*(\d+)/);
		if (!match) return null;
		return { current: parseInt(match[1], 10), total: parseInt(match[2], 10) };
	}
</script>

<!-- Stats Row -->
<div class="stats-row">
	<div class="stat-card">
		<div class="stat-icon icon-green"><Server size={24} /></div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{stats.active_servers}</span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Active Servers</span>
		</div>
	</div>
	<div class="stat-card">
		<div class="stat-icon icon-red"><ServerOff size={24} /></div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{stats.inactive_servers}</span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Inactive</span>
		</div>
	</div>
	<div class="stat-card">
		<div class="stat-icon icon-blue"><Activity size={24} /></div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{formatNumber(stats.total_pings)}</span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Total Pings</span>
		</div>
	</div>
	<div class="stat-card">
		<div class="stat-icon icon-purple"><Users size={24} /></div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{stats.users}</span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Users</span>
		</div>
	</div>
	<div class="stat-card">
		<div class="stat-icon icon-orange"><Database size={24} /></div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{formatBytes(stats.database_bytes)}</span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Database</span>
		</div>
	</div>
	<div class="stat-card">
		<div
			class="stat-icon"
			class:icon-green={stats?.running_jobs === 0}
			class:icon-blue={stats && stats.running_jobs > 0}
		>
			<Zap size={24} />
		</div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number"
					>{stats.running_jobs}<span class="stat-sub">/{stats.total_jobs}</span></span
				>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Jobs Running</span>
		</div>
	</div>
</div>

<!-- Jobs Table -->
<div class="table-card">
	{#if loading}
		<div class="empty-state">
			<Loader size={20} class="spin" />
			<span>Loading jobs...</span>
		</div>
	{:else}
		<table class="jobs-table">
			<thead>
				<tr>
					<th></th>
					<th>Job</th>
					<th>Schedule</th>
					<th>Last Run</th>
					<th>Duration</th>
					<th>Status</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each jobs as job}
					<tr class="job-row" class:running={job.status === 'running'}>
						<td class="cell-icon">
							{#if job.status === 'running'}
								<Loader size={14} class="spin" />
							{:else if job.status === 'success'}
								<CheckCircle size={14} />
							{:else if job.status === 'failed'}
								<XCircle size={14} />
							{:else}
								<Clock size={14} />
							{/if}
						</td>
						<td class="cell-name">
							<button class="name-btn" onclick={() => toggleLogs(job.name)}>
								{job.name}
							</button>
							<span class="job-desc">{job.description}</span>
							{#if job.status === 'running' && job.progress}
								{@const parsed = parseProgress(job.progress)}
								{#if parsed}
									<div class="progress-pill">
										<div
											class="progress-pill-fill"
											style="width: {(parsed.current / parsed.total) * 100}%"
										></div>
										<span class="progress-pill-label">{parsed.current}/{parsed.total}</span>
									</div>
								{:else}
									<span class="progress-text">{job.progress}</span>
								{/if}
							{/if}
						</td>
						<td class="cell-schedule">{job.schedule}</td>
						<td class="cell-time">
							{#if job.schedule === 'manual'}
								{timeAgo(job.last_run)}
							{:else}
								{timeAgo(job.last_run)}
								{#if job.status !== 'running'}
									<span class="next-run">next {timeUntil(job.next_run)}</span>
								{/if}
							{/if}
						</td>
						<td class="cell-duration">{formatDuration(job.last_duration_seconds)}</td>
						<td class="cell-status">
							{#if job.status === 'running'}
								<span class="status-badge running">running</span>
							{:else if job.last_error}
								<span class="status-badge failed" title={job.last_error}>failed</span>
							{:else if job.status === 'success'}
								<span class="status-badge success">ok</span>
							{/if}
						</td>
						<td class="cell-action">
							{#if job.status === 'running'}
								<div class="action-btn busy"><Loader size={12} class="spin" /></div>
							{:else}
								<button class="action-btn" onclick={() => triggerJob(job.name)} title="Run now">
									<Play size={12} />
								</button>
							{/if}
						</td>
					</tr>
					{#if expandedJob === job.name}
						<tr class="logs-row">
							<td colspan="7">
								<div class="logs-panel">
									{#if logsLoading}
										<span class="logs-loading"><Loader size={12} class="spin" /> Loading...</span>
									{:else if jobLogs.length === 0}
										<span class="logs-empty">No history yet</span>
									{:else}
										<table class="inner-table">
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
													<tr class:log-err={log.status === 'failed'}>
														<td>{formatTime(log.started_at)}</td>
														<td><span class="trigger-badge {log.trigger}">{log.trigger}</span></td>
														<td><span class="status-badge {log.status}">{log.status}</span></td>
														<td>{formatDuration(log.duration_seconds)}</td>
														<td class="cell-summary">{log.error || log.summary || '—'}</td>
													</tr>
												{/each}
											</tbody>
										</table>
									{/if}
								</div>
							</td>
						</tr>
					{/if}
				{/each}
			</tbody>
		</table>
	{/if}
</div>

<style>
	/* Stats */
	.stats-row {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 0.75rem;
		margin-bottom: 1rem;
	}

	.stat-card {
		display: flex;
		align-items: center;
		gap: 0.625rem;
		background: #202020;
		border-radius: 0.5rem;
		padding: 0.75rem 1rem;
	}

	.stat-icon {
		color: #6b7280;
		flex-shrink: 0;
	}
	.icon-green {
		color: #22c55e;
	}
	.icon-red {
		color: #ef4444;
	}
	.icon-blue {
		color: #3b82f6;
	}
	.icon-purple {
		color: #a855f7;
	}
	.icon-orange {
		color: #f59e0b;
	}

	.stat-body {
		display: flex;
		flex-direction: column;
	}

	.stat-number {
		font-size: 1.125rem;
		font-weight: 600;
		line-height: 1.2;
		color: #e5e7eb;
		font-variant-numeric: tabular-nums;
	}

	.stat-sub {
		font-size: 0.75rem;
		font-weight: 400;
		color: #6b7280;
	}

	.stat-label {
		font-size: 0.6875rem;
		font-weight: 500;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		white-space: nowrap;
	}

	.stat-skeleton {
		display: block;
		height: 1.125rem;
		width: 2rem;
		background: #2b2b2b;
		border-radius: 0.25rem;
		animation: pulse 2s ease-in-out infinite;
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

	/* Table */
	.table-card {
		background: #202020;
		border-radius: 0.5rem;
		padding: 0 1rem 1rem;
	}

	.empty-state {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 3rem 0;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.jobs-table {
		width: 100%;
		border-collapse: separate;
		border-spacing: 0 0.25rem;
	}

	.jobs-table thead tr {
		color: #9ca3af;
	}

	.jobs-table th {
		text-align: left;
		padding: 0.5rem;
		font-size: 0.75rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.025em;
	}

	.job-row td {
		padding: 0.5rem;
		transition: background-color 0.15s ease;
	}

	.job-row:hover td {
		background: #2b2b2b;
	}
	.job-row:hover td:first-child {
		border-radius: 0.5rem 0 0 0.5rem;
	}
	.job-row:hover td:last-child {
		border-radius: 0 0.5rem 0.5rem 0;
	}

	/* Icon cell */
	.cell-icon {
		width: 1px;
	}
	.job-row :global(.lucide-check-circle) {
		color: #22c55e;
	}
	.job-row :global(.lucide-x-circle) {
		color: #ef4444;
	}
	.job-row :global(.lucide-clock) {
		color: #4b5563;
	}
	.job-row :global(.lucide-loader) {
		color: #3b82f6;
	}

	:global(.spin) {
		animation: spin 1s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	/* Name cell */
	.cell-name {
		min-width: 200px;
	}

	.name-btn {
		display: block;
		font-size: 0.8125rem;
		font-weight: 500;
		color: #e5e7eb;
		background: none;
		border: none;
		padding: 0;
		cursor: pointer;
		text-align: left;
		font-family: ui-monospace, SFMono-Regular, monospace;
	}
	.name-btn:hover {
		color: #3b82f6;
	}

	.job-desc {
		display: block;
		font-size: 0.6875rem;
		color: #6b7280;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.progress-text {
		display: block;
		font-size: 0.6875rem;
		color: #3b82f6;
	}

	.progress-pill {
		position: relative;
		display: inline-flex;
		align-items: center;
		width: 100px;
		height: 18px;
		background: #2b2b2b;
		border-radius: 9999px;
		overflow: hidden;
		margin-top: 0.25rem;
	}

	.progress-pill-fill {
		position: absolute;
		inset: 0;
		background: linear-gradient(90deg, #3b82f6, #60a5fa);
		border-radius: 9999px;
		transition: width 0.3s ease;
	}

	.progress-pill-label {
		position: relative;
		z-index: 1;
		width: 100%;
		text-align: center;
		font-size: 0.625rem;
		font-weight: 600;
		color: #fff;
		text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
	}

	/* Other cells */
	.cell-schedule {
		font-size: 0.75rem;
		color: #9ca3af;
		white-space: nowrap;
	}

	.cell-time {
		font-size: 0.75rem;
		color: #9ca3af;
		white-space: nowrap;
	}

	.next-run {
		display: block;
		font-size: 0.625rem;
		color: #4b5563;
	}

	.cell-duration {
		font-size: 0.75rem;
		color: #6b7280;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}

	.cell-status {
		white-space: nowrap;
	}

	.cell-action {
		width: 1px;
	}

	/* Action button */
	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		border-radius: 0.375rem;
		border: 1px solid #333;
		background: #252525;
		color: #22c55e;
		cursor: pointer;
		transition: all 0.15s;
	}
	.action-btn:hover {
		background: rgba(34, 197, 94, 0.15);
		border-color: rgba(34, 197, 94, 0.3);
	}
	.action-btn.busy {
		color: #3b82f6;
		border-color: rgba(59, 130, 246, 0.3);
		background: rgba(59, 130, 246, 0.1);
		cursor: default;
	}

	/* Badges */
	.status-badge,
	.trigger-badge {
		display: inline-block;
		padding: 0.0625rem 0.375rem;
		border-radius: 9999px;
		font-size: 0.625rem;
		font-weight: 500;
	}
	.status-badge.success {
		background: rgba(34, 197, 94, 0.15);
		color: #22c55e;
	}
	.status-badge.failed {
		background: rgba(239, 68, 68, 0.15);
		color: #ef4444;
	}
	.status-badge.running {
		background: rgba(59, 130, 246, 0.15);
		color: #3b82f6;
	}
	.status-badge.idle {
		background: rgba(107, 114, 128, 0.15);
		color: #6b7280;
	}
	.trigger-badge.manual {
		background: rgba(168, 85, 247, 0.15);
		color: #a855f7;
	}
	.trigger-badge.scheduled {
		background: rgba(107, 114, 128, 0.15);
		color: #6b7280;
	}

	/* Logs */
	.logs-row td {
		padding: 0 !important;
	}

	.logs-panel {
		background: #1a1a1a;
		border-top: 1px solid #2a2a2a;
		padding: 0.75rem 1rem;
	}

	.logs-loading,
	.logs-empty {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #4b5563;
		font-size: 0.75rem;
	}

	.inner-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.6875rem;
	}

	.inner-table th {
		text-align: left;
		color: #6b7280;
		font-weight: 500;
		padding: 0.25rem 0.5rem;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		border-bottom: 1px solid #2a2a2a;
	}

	.inner-table td {
		padding: 0.375rem 0.5rem;
		color: #9ca3af;
		border-bottom: 1px solid #1f1f1f;
	}

	.cell-summary {
		max-width: 300px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	tr.log-err td {
		color: #f87171;
	}

	/* Responsive */
	@media (max-width: 900px) {
		.stats-row {
			grid-template-columns: repeat(3, 1fr);
		}
	}
	@media (max-width: 550px) {
		.stats-row {
			grid-template-columns: repeat(2, 1fr);
		}
	}
</style>
