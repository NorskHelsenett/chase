<script>
	import { AlertTriangle, Lock, Info, Shield, ShieldAlert, ArrowRight } from 'lucide-svelte';
	import { fade, slide } from 'svelte/transition';
	import ChecksGrid from './ChecksGrid.svelte';

	export let loading = false;
	export let results = {};

	let expandedIssue = null;

	function getRiskClass(risk) {
		switch (risk) {
			case 'CRITICAL': return 'risk-critical';
			case 'HIGH': return 'risk-high';
			case 'MEDIUM': return 'risk-medium';
			case 'LOW': return 'risk-low';
			default: return 'risk-info';
		}
	}

	function getScoreClass(score) {
		switch (score) {
			case 'A+': return 'score-a-plus';
			case 'A': return 'score-a';
			case 'B': return 'score-b';
			case 'C': return 'score-c';
			case 'D': return 'score-d';
			case 'E':
			case 'F': return 'score-f';
			default: return 'score-none';
		}
	}
</script>

{#if loading}
	<div class="card loading">
		<div class="card-header">
			<Lock size={20} />
			<div class="skeleton skeleton-title"></div>
		</div>
		<div class="skeleton-content">
			<div class="skeleton skeleton-large"></div>
			{#each Array(3) as _}
				<div class="skeleton skeleton-line"></div>
			{/each}
		</div>
	</div>
{:else if results?.headers}
	<div class="card" in:fade={{ duration: 200 }}>
		<!-- Header -->
		<div class="card-header">
			<Lock size={20} />
			<h2>Security Headers Analysis</h2>
		</div>

		<!-- Score -->
		<div class="score-section">
			<div class="score-value {getScoreClass(results.headers.score)}">
				{results.headers.score}
			</div>
			<div class="score-label">Security Headers Score</div>
		</div>

		<!-- Issues -->
		{#if results.headers.issues.length > 0}
			<div class="section">
				<h3 class="section-title error">
					<AlertTriangle size={20} />
					<span>Security Issues</span>
				</h3>

				<div class="issues-list">
					{#each results.headers.issues as issue, index}
						<div class="issue-card">
							<button
								class="issue-header"
								on:click={() => (expandedIssue = expandedIssue === index ? null : index)}
							>
								<ShieldAlert size={20} class="issue-icon {getRiskClass(issue.risk)}" />
								<div class="issue-content">
									<span class="issue-description">{issue.description}</span>
									<span class="risk-badge {getRiskClass(issue.risk)}">
										{issue.risk}
									</span>
								</div>
							</button>

							{#if expandedIssue === index}
								<div class="issue-details" transition:slide|local>
									<div class="detail-box evidence">
										<div class="detail-header">
											<Info size={16} />
											<span>Current Configuration</span>
										</div>
										<pre class="detail-content">{issue.evidence}</pre>
									</div>

									<div class="detail-box mitigation">
										<div class="detail-header">
											<Shield size={16} />
											<span>Recommended Action</span>
										</div>
										<div class="detail-content with-arrow">
											<ArrowRight size={16} />
											<span>{issue.mitigation}</span>
										</div>
									</div>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/if}

		{#if results.headers.cookieFindings?.length > 0}
			<div class="section">
				<h3 class="section-title warning">
					<AlertTriangle size={20} />
					<span>Cookie Findings</span>
				</h3>
				<div class="findings-list">
					{#each results.headers.cookieFindings as finding}
						<div class="finding-card">
							<div class="finding-description">{finding.description}</div>
							<div class="finding-mitigation">{finding.mitigation}</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<div class="section">
			<h3 class="section-title info">
				<Shield size={20} />
				<span>Cross-Origin Policy</span>
			</h3>
			{#if results.headers.corsFindings?.length > 0}
				<div class="findings-list">
					{#each results.headers.corsFindings as finding}
						<div class="finding-card">
							<div class="finding-description">{finding.description}</div>
							<div class="finding-mitigation">{finding.mitigation}</div>
						</div>
					{/each}
				</div>
			{:else}
				<div class="success-message">
					No cross-origin issues detected—CORS policy appears restrictive.
				</div>
			{/if}
		</div>

		<ChecksGrid checks={results.headers.passed} />
	</div>
{/if}

<style>
	.card {
		width: 100%;
		background: #202020;
		border-radius: 0.5rem;
		padding: 1.5rem;
	}

	.card.loading {
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	.card-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
		color: #e5e7eb;
	}

	.card-header h2 {
		font-size: 1.25rem;
		font-weight: 500;
	}

	.skeleton {
		background: #374151;
		border-radius: 0.25rem;
	}

	.skeleton-title {
		height: 2rem;
		width: 33%;
	}

	.skeleton-content {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.skeleton-large {
		height: 3rem;
	}

	.skeleton-line {
		height: 1rem;
		width: 100%;
	}

	.score-section {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin-bottom: 2rem;
	}

	.score-value {
		font-size: 2.5rem;
		font-weight: 700;
	}

	.score-label {
		color: #9ca3af;
	}

	.score-a-plus { color: #10b981; }
	.score-a { color: #22c55e; }
	.score-b { color: #3b82f6; }
	.score-c { color: #eab308; }
	.score-d { color: #f97316; }
	.score-f { color: #ef4444; }
	.score-none { color: #6b7280; }

	.section {
		margin-bottom: 2rem;
	}

	.section:last-child {
		margin-bottom: 0;
	}

	.section-title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 1.125rem;
		font-weight: 600;
		margin-bottom: 0.75rem;
	}

	.section-title.error { color: #f87171; }
	.section-title.warning { color: #fbbf24; }
	.section-title.info { color: #60a5fa; }
	.section-title.success { color: #4ade80; }

	.issues-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.issue-card {
		background: #2b2b2b;
		border-radius: 0.5rem;
		overflow: hidden;
	}

	.issue-header {
		width: 100%;
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		padding: 0.75rem 1rem;
		text-align: left;
		background: transparent;
		border: none;
		cursor: pointer;
		color: #e5e7eb;
	}

	.issue-header:hover {
		background: #333;
	}

	:global(.issue-icon) {
		flex-shrink: 0;
		margin-top: 0.125rem;
	}

	:global(.issue-icon.risk-critical) { color: #ef4444; }
	:global(.issue-icon.risk-high) { color: #f97316; }
	:global(.issue-icon.risk-medium) { color: #eab308; }
	:global(.issue-icon.risk-low) { color: #22c55e; }
	:global(.issue-icon.risk-info) { color: #3b82f6; }

	.issue-content {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.issue-description {
		font-weight: 500;
	}

	.risk-badge {
		display: inline-flex;
		padding: 0.125rem 0.5rem;
		font-size: 0.6875rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.025em;
		border-radius: 9999px;
	}

	.risk-badge.risk-critical {
		color: #ef4444;
		background: rgba(239, 68, 68, 0.1);
	}

	.risk-badge.risk-high {
		color: #f97316;
		background: rgba(249, 115, 22, 0.1);
	}

	.risk-badge.risk-medium {
		color: #eab308;
		background: rgba(234, 179, 8, 0.1);
	}

	.risk-badge.risk-low {
		color: #22c55e;
		background: rgba(34, 197, 94, 0.1);
	}

	.risk-badge.risk-info {
		color: #3b82f6;
		background: rgba(59, 130, 246, 0.1);
	}

	.issue-details {
		padding: 0 1rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.detail-box {
		padding: 0.75rem;
		border-radius: 0.5rem;
		border: 1px solid;
	}

	.detail-box.evidence {
		background: rgba(239, 68, 68, 0.1);
		border-color: rgba(127, 29, 29, 0.5);
		color: #fca5a5;
	}

	.detail-box.mitigation {
		background: rgba(59, 130, 246, 0.1);
		border-color: rgba(30, 58, 138, 0.5);
		color: #93c5fd;
	}

	.detail-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		font-weight: 600;
		margin-bottom: 0.5rem;
	}

	.detail-content {
		font-size: 0.875rem;
		white-space: pre-wrap;
		font-family: ui-monospace, monospace;
		margin: 0;
	}

	.detail-content.with-arrow {
		display: flex;
		gap: 0.5rem;
		font-family: inherit;
	}

	.findings-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.finding-card {
		padding: 0.75rem;
		background: #2b2b2b;
		border-radius: 0.5rem;
	}

	.finding-description {
		font-weight: 500;
		color: #e5e7eb;
	}

	.finding-mitigation {
		font-size: 0.875rem;
		color: #9ca3af;
		margin-top: 0.25rem;
	}

	.success-message {
		font-size: 0.875rem;
		color: #4ade80;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}
</style>
