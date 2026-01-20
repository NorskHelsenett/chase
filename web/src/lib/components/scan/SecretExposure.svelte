<script>
	import { Key, AlertTriangle, Shield, CheckCircle, XCircle, FileCode, Eye } from 'lucide-svelte';
	import { fade, slide } from 'svelte/transition';
	import ChecksGrid from './ChecksGrid.svelte';

	export let loading = false;
	export let results = {};

	let expandedFinding = null;

	function getRiskClass(risk) {
		switch (risk?.toUpperCase()) {
			case 'CRITICAL':
				return 'risk-critical';
			case 'HIGH':
				return 'risk-high';
			case 'MEDIUM':
				return 'risk-medium';
			case 'LOW':
				return 'risk-low';
			default:
				return 'risk-info';
		}
	}

	function getSourceLabel(source) {
		if (source === 'document') return 'HTML Document';
		if (source.startsWith('inline-script')) {
			const num = source.match(/\d+/)?.[0];
			return num ? `Inline Script #${num}` : 'Inline Script';
		}
		if (source.startsWith('external-script')) {
			return source.replace('external-script:', 'External: ');
		}
		return source;
	}

	// Combine checks and sources into a single array for ChecksGrid
	$: allChecks = (() => {
		const checks = (results?.secretExposure?.checks || []);
		if (checks.length > 0) return checks;
		// Fallback to sources if no checks provided
		return (results?.secretExposure?.sources || []).map(s => ({ name: getSourceLabel(s), passed: true }));
	})();
</script>

{#if loading}
	<div class="card loading">
		<div class="card-header">
			<Key size={18} />
			<div class="skeleton skeleton-title"></div>
		</div>
		<div class="skeleton-content">
			<div class="skeleton skeleton-large"></div>
			{#each Array(2) as _}
				<div class="skeleton skeleton-line"></div>
			{/each}
		</div>
	</div>
{:else if results?.secretExposure}
	<div class="card" in:fade={{ duration: 200 }}>
		<!-- Header -->
		<div class="card-header">
			<Key size={18} />
			<h2>Secret Exposure Analysis</h2>
		</div>

		<!-- Risk Level -->
		<div class="risk-section">
			<div class="risk-value {getRiskClass(results.secretExposure.risk)}">
				{results.secretExposure.risk || 'N/A'}
			</div>
			<div class="risk-label">Exposure Risk</div>
		</div>

		<!-- Findings Grid -->
		{#if results.secretExposure.findings?.length > 0}
			<div class="section">
				<h3 class="section-title error">
					<AlertTriangle size={14} />
					<span>Exposed Secrets</span>
				</h3>

				<div class="findings-grid">
					{#each results.secretExposure.findings as finding, index}
						<button
							class="check-item clickable"
							class:expanded={expandedFinding === index}
							on:click={() => (expandedFinding = expandedFinding === index ? null : index)}
						>
							<XCircle size={14} class="check-icon {getRiskClass(finding.severity || 'HIGH')}" />
							<span class="check-name failed">{finding.type || 'Secret Detected'}</span>
							<span class="severity-badge {getRiskClass(finding.severity || 'HIGH')}">
								{finding.severity || 'HIGH'}
							</span>
						</button>

						{#if expandedFinding === index}
							<div class="finding-details" transition:slide|local>
								{#if finding.source}
									<div class="detail-box source">
										<div class="detail-header">
											<FileCode size={12} />
											<span>Source</span>
										</div>
										<div class="detail-content">{finding.source}</div>
									</div>
								{/if}

								{#if finding.evidence}
									<div class="detail-box evidence">
										<div class="detail-header">
											<Eye size={12} />
											<span>Evidence</span>
										</div>
										<pre class="detail-content mono">{finding.evidence}</pre>
									</div>
								{/if}

								{#if finding.description}
									<div class="detail-box info">
										<div class="detail-header">
											<AlertTriangle size={12} />
											<span>Description</span>
										</div>
										<div class="detail-content">{finding.description}</div>
									</div>
								{/if}

								{#if finding.mitigation}
									<div class="detail-box mitigation">
										<div class="detail-header">
											<Shield size={12} />
											<span>Recommended Action</span>
										</div>
										<div class="detail-content">{finding.mitigation}</div>
									</div>
								{/if}
							</div>
						{/if}
					{/each}
				</div>
			</div>
		{/if}

		<!-- Security Checks Grid -->
		<ChecksGrid checks={allChecks} />
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
		margin-bottom: 1rem;
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
		height: 1.5rem;
		width: 33%;
	}

	.skeleton-content {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.skeleton-large {
		height: 2rem;
	}

	.skeleton-line {
		height: 0.75rem;
		width: 100%;
	}

	.risk-section {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 1rem;
	}

	.risk-value {
		font-size: 1.5rem;
		font-weight: 700;
		text-transform: uppercase;
	}

	.risk-label {
		color: #9ca3af;
		font-size: 0.8125rem;
	}

	.risk-critical {
		color: #ef4444;
	}
	.risk-high {
		color: #f97316;
	}
	.risk-medium {
		color: #eab308;
	}
	.risk-low {
		color: #22c55e;
	}
	.risk-info {
		color: #3b82f6;
	}

	.section {
		margin-bottom: 1rem;
	}

	.section:last-child {
		margin-bottom: 0;
	}

	.section-title {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.8125rem;
		font-weight: 600;
		margin-bottom: 0.5rem;
		color: #9ca3af;
	}

	.section-title.error {
		color: #f87171;
	}

	.findings-grid {
		display: grid;
		grid-template-columns: repeat(2, max-content);
		gap: 0.25rem 2rem;
	}

	@media (min-width: 640px) {
		.findings-grid {
			grid-template-columns: repeat(3, max-content);
		}
	}

	@media (min-width: 900px) {
		.findings-grid {
			grid-template-columns: repeat(4, max-content);
		}
	}

	.check-item {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0;
		font-size: 0.8125rem;
		color: #d1d5db;
		white-space: nowrap;
		background: none;
		border: none;
		cursor: pointer;
		transition: color 0.15s ease;
	}

	.check-item:hover {
		color: #fff;
	}

	.check-item:hover .check-name {
		text-decoration: underline;
	}

	:global(.check-icon.risk-critical) {
		color: #ef4444;
	}
	:global(.check-icon.risk-high) {
		color: #f97316;
	}
	:global(.check-icon.risk-medium) {
		color: #eab308;
	}
	:global(.check-icon.risk-low) {
		color: #22c55e;
	}

	.check-name.failed {
		color: #fca5a5;
		font-weight: 500;
	}

	.severity-badge {
		display: inline-flex;
		padding: 0.0625rem 0.375rem;
		font-size: 0.5625rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.025em;
		border-radius: 9999px;
		margin-left: auto;
	}

	.severity-badge.risk-critical {
		color: #ef4444;
		background: rgba(239, 68, 68, 0.1);
	}

	.severity-badge.risk-high {
		color: #f97316;
		background: rgba(249, 115, 22, 0.1);
	}

	.severity-badge.risk-medium {
		color: #eab308;
		background: rgba(234, 179, 8, 0.1);
	}

	.severity-badge.risk-low {
		color: #22c55e;
		background: rgba(34, 197, 94, 0.1);
	}

	.finding-details {
		grid-column: 1 / -1;
		padding: 0.75rem;
		margin-top: 0.25rem;
		background: #2b2b2b;
		border-radius: 0.375rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.detail-box {
		padding: 0.5rem;
		border-radius: 0.375rem;
		border: 1px solid;
	}

	.detail-box.source {
		background: rgba(59, 130, 246, 0.05);
		border-color: rgba(30, 58, 138, 0.3);
		color: #93c5fd;
	}

	.detail-box.evidence {
		background: rgba(239, 68, 68, 0.1);
		border-color: rgba(127, 29, 29, 0.5);
		color: #fca5a5;
	}

	.detail-box.info {
		background: rgba(234, 179, 8, 0.1);
		border-color: rgba(133, 77, 14, 0.5);
		color: #fde047;
	}

	.detail-box.mitigation {
		background: rgba(34, 197, 94, 0.1);
		border-color: rgba(22, 101, 52, 0.5);
		color: #86efac;
	}

	.detail-header {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.75rem;
		font-weight: 600;
		margin-bottom: 0.25rem;
	}

	.detail-content {
		font-size: 0.75rem;
	}

	.detail-content.mono {
		font-family: ui-monospace, monospace;
		white-space: pre-wrap;
		word-break: break-all;
	}

	.success-message {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #4ade80;
		font-size: 0.8125rem;
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
</style>
