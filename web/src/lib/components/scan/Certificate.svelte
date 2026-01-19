<script>
	import {
		Wrench,
		AlertTriangle,
		Shield,
		Calendar,
		Building2,
		Award,
		Key,
		Hash
	} from 'lucide-svelte';
	import { fade } from 'svelte/transition';

	export let loading = true;
	export let results = {};

	function getGradeClass(grade) {
		switch (grade) {
			case 'A+': return 'grade-a-plus';
			case 'A': return 'grade-a';
			case 'B': return 'grade-b';
			case 'C': return 'grade-c';
			case 'D': return 'grade-d';
			case 'F': return 'grade-f';
			default: return 'grade-none';
		}
	}
</script>

<section class="cert-section">
	<h2 class="section-header">
		<Wrench size={20} />
		<span>SSL/TLS Certificate Analysis</span>
	</h2>

	{#if loading}
		<div class="card loading">
			<div class="skeleton skeleton-title"></div>
			<div class="skeleton-content">
				{#each Array(4) as _}
					<div class="skeleton skeleton-line"></div>
				{/each}
			</div>
		</div>
	{:else if results}
		<div class="card" in:fade={{ duration: 200 }}>
			<!-- Grade Section -->
			<div class="grade-section">
				<div class="grade-value {getGradeClass(results.certificate.grade)}">
					{results.certificate.grade}
				</div>
				<div class="grade-label">Certificate Grade</div>
			</div>

			<div class="content-grid">
				<!-- Certificate Details -->
				<div class="details-column">
					<h3 class="subsection-header">
						<Shield size={16} />
						<span>Certificate Details</span>
					</h3>

					<div class="details-list">
						<!-- Validity Period -->
						<div class="detail-group">
							<div class="detail-group-header">
								<Calendar size={16} />
								<h4>Validity Period</h4>
							</div>
							<dl class="detail-grid">
								<dt>Valid from:</dt>
								<dd>{new Date(results.certificate.validFrom).toLocaleString(undefined, {
									dateStyle: 'medium',
									timeStyle: 'short'
								})}</dd>
								<dt>Valid until:</dt>
								<dd>{new Date(results.certificate.validUntil).toLocaleString(undefined, {
									dateStyle: 'medium',
									timeStyle: 'short'
								})}</dd>
							</dl>
						</div>

						<!-- Organization Info -->
						<div class="detail-group">
							<div class="detail-group-header">
								<Building2 size={16} />
								<h4>Organization Info</h4>
							</div>
							<dl class="detail-grid">
								<dt>Organization:</dt>
								<dd>{results.certificate.organization || 'Unknown'}</dd>
								<dt>Issuer:</dt>
								<dd>{results.certificate.issuer || 'Unknown'}</dd>
							</dl>
						</div>

						<!-- Technical Details -->
						<div class="detail-group">
							<div class="detail-group-header">
								<Key size={16} />
								<h4>Technical Details</h4>
							</div>
							<dl class="detail-grid">
								<dt>Key Type:</dt>
								<dd>{results.certificate.publicKeyType} ({results.certificate.publicKeyBits} bits)</dd>
								<dt>Signature:</dt>
								<dd>{results.certificate.signatureAlg || 'Unknown'}</dd>
								<dt>Serial:</dt>
								<dd class="mono">{results.certificate.serialNumber || 'Unknown'}</dd>
							</dl>
						</div>

						<!-- Subject DNS Names -->
						{#if results.certificate.subjectDNS && results.certificate.subjectDNS.length > 0}
							<div class="detail-group">
								<div class="detail-group-header">
									<Hash size={16} />
									<h4>Protected Domains</h4>
								</div>
								<ul class="dns-list">
									{#each results.certificate.subjectDNS as dns}
										<li>• {dns}</li>
									{/each}
								</ul>
							</div>
						{/if}
					</div>
				</div>

				<!-- Findings and Warnings -->
				<div class="findings-column">
					{#if results.certificate.findings.length > 0}
						<div class="findings-group">
							<h3 class="subsection-header">
								<Award size={16} />
								<span>Findings</span>
							</h3>
							<ul class="findings-list">
								{#each results.certificate.findings as finding}
									<li class="finding-item">
										<div class="finding-description">{finding.description}</div>
										<div class="finding-evidence">{finding.evidence}</div>
									</li>
								{/each}
							</ul>
						</div>
					{/if}

					{#if results.certificate.warnings.length > 0}
						<div class="findings-group">
							<h3 class="subsection-header warning">
								<AlertTriangle size={16} />
								<span>Warnings</span>
							</h3>
							<ul class="findings-list">
								{#each results.certificate.warnings as warning}
									<li class="finding-item">
										<div class="finding-description">{warning.description}</div>
										<div class="finding-evidence">{warning.evidence}</div>
									</li>
								{/each}
							</ul>
						</div>
					{/if}

					{#if results.certificate.tlsVersions.length > 0}
						<div class="findings-group">
							<h3 class="subsection-header">
								<Shield size={16} />
								<span>TLS Versions</span>
							</h3>
							<ul class="tls-list">
								{#each results.certificate.tlsVersions as version}
									<li>• {version}</li>
								{/each}
							</ul>
						</div>
					{/if}
				</div>
			</div>
		</div>
	{/if}
</section>

<style>
	.cert-section {
		width: 100%;
	}

	.section-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 1.25rem;
		font-weight: 500;
		color: #e5e7eb;
		margin-bottom: 1rem;
	}

	.card {
		background: #202020;
		border-radius: 0.5rem;
		padding: 1.5rem;
	}

	.card.loading {
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	.skeleton {
		background: #374151;
		border-radius: 0.25rem;
	}

	.skeleton-title {
		height: 2rem;
		width: 25%;
		margin-bottom: 1rem;
	}

	.skeleton-content {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.skeleton-line {
		height: 1rem;
		width: 100%;
	}

	.grade-section {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.grade-value {
		font-size: 2.5rem;
		font-weight: 700;
	}

	.grade-label {
		color: #9ca3af;
	}

	.grade-a-plus { color: #10b981; }
	.grade-a { color: #22c55e; }
	.grade-b { color: #3b82f6; }
	.grade-c { color: #eab308; }
	.grade-d { color: #f97316; }
	.grade-f { color: #ef4444; }
	.grade-none { color: #6b7280; }

	.content-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.5rem;
	}

	.subsection-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #9ca3af;
		font-size: 0.875rem;
		font-weight: 500;
		margin-bottom: 1rem;
	}

	.subsection-header.warning {
		color: #fbbf24;
	}

	.details-list {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.detail-group {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.detail-group-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #9ca3af;
	}

	.detail-group-header h4 {
		font-size: 0.875rem;
		font-weight: 400;
	}

	.detail-grid {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.25rem 0.75rem;
		font-size: 0.875rem;
	}

	.detail-grid dt {
		color: #9ca3af;
	}

	.detail-grid dd {
		color: #d1d5db;
		margin: 0;
	}

	.detail-grid dd.mono {
		font-family: ui-monospace, monospace;
		font-size: 0.75rem;
	}

	.dns-list {
		list-style: none;
		padding: 0;
		margin: 0;
		font-size: 0.875rem;
		color: #d1d5db;
		font-family: ui-monospace, monospace;
	}

	.findings-column {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.findings-group {
		display: flex;
		flex-direction: column;
	}

	.findings-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.finding-item {
		font-size: 0.875rem;
	}

	.finding-description {
		color: #d1d5db;
		font-weight: 500;
	}

	.finding-evidence {
		color: #9ca3af;
		font-size: 0.75rem;
		margin-top: 0.25rem;
	}

	.tls-list {
		list-style: none;
		padding: 0;
		margin: 0;
		font-size: 0.875rem;
		color: #d1d5db;
	}

	.tls-list li {
		padding: 0.125rem 0;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	@media (max-width: 768px) {
		.content-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
