<script>
	import { Clock, Globe, Shield, Lock, Server, FileText, AlertTriangle, CheckCircle, XCircle, ExternalLink, Activity, Heart, File, Mail, Link, Key, Eye, EyeOff, ChevronDown, ChevronUp } from 'lucide-svelte';
	import { fade, fly, slide } from 'svelte/transition';
	import { onMount, onDestroy } from 'svelte';
	import { searchHistory } from '$lib/stores/searchStore';
	import { getRelativeTime } from '$lib/utils/time';

	/** @type {import('./$types').PageData} */
	export let data;

	let loading = true;
	let results = null;
	let searchTimestamp = Date.now();
	let relativeTime = 'now';
	let timeInterval;
	let screenshotLoaded = false;
	let expandedSections = {
		securityTxt: false,
		robotsTxt: false,
		healthProbes: false
	};

	function updateRelativeTime() {
		relativeTime = getRelativeTime(searchTimestamp);
	}

	onMount(() => {
		timeInterval = setInterval(updateRelativeTime, 1000);
	});

	onDestroy(() => {
		if (timeInterval) clearInterval(timeInterval);
	});

	async function fetchSearchResults(query) {
		loading = true;
		searchTimestamp = Date.now();
		screenshotLoaded = false;
		try {
			const response = await fetch(`/api/security/${encodeURIComponent(query)}`);
			const data = await response.json();
			results = data;
			searchHistory.addSearch(query, data);
		} catch (error) {
			console.error('Error fetching search results:', error);
			results = null;
		} finally {
			loading = false;
		}
	}

	$: if (data.query) {
		fetchSearchResults(data.query);
	}

	// Calculate combined grade
	function calculateOverallGrade(results) {
		if (!results) return 'N/A';

		const gradeValues = { 'A+': 10, 'A': 9, 'B': 7, 'C': 5, 'D': 3, 'E': 2, 'F': 1 };
		const riskValues = { 'LOW': 9, 'MEDIUM': 5, 'HIGH': 2, 'CRITICAL': 1 };

		let scores = [];

		if (results.headers?.score) scores.push(gradeValues[results.headers.score] || 5);
		if (results.certificate?.grade) scores.push(gradeValues[results.certificate.grade] || 5);
		if (results.adminPages?.risk) scores.push(riskValues[results.adminPages.risk] || 5);
		if (results.swagger?.risk) scores.push(riskValues[results.swagger.risk] || 5);
		if (results.infrastructure?.risk) scores.push(riskValues[results.infrastructure.risk] || 5);

		if (scores.length === 0) return 'N/A';

		const avg = scores.reduce((a, b) => a + b, 0) / scores.length;

		if (avg >= 9) return 'A';
		if (avg >= 7) return 'B';
		if (avg >= 5) return 'C';
		if (avg >= 3) return 'D';
		return 'F';
	}

	function getGradeClass(grade) {
		switch (grade) {
			case 'A+':
			case 'A': return 'grade-a';
			case 'B': return 'grade-b';
			case 'C': return 'grade-c';
			case 'D': return 'grade-d';
			default: return 'grade-f';
		}
	}

	function getDaysUntil(dateStr) {
		if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return null;
		const diff = new Date(dateStr) - new Date();
		return Math.floor(diff / (1000 * 60 * 60 * 24));
	}

	function formatDate(dateStr) {
		if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return 'N/A';
		return new Date(dateStr).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	function getIssueCount(results) {
		let count = 0;
		if (results?.headers?.issues) count += results.headers.issues.length;
		if (results?.certificate?.warnings) count += results.certificate.warnings.length;
		if (results?.dnsRecords?.findings) count += results.dnsRecords.findings.length;
		if (results?.infrastructure?.findings) count += results.infrastructure.findings.length;
		return count;
	}

	function getPassedCount(results) {
		let count = 0;
		if (results?.headers?.passed) count += results.headers.passed.length;
		if (results?.certificate?.findings) count += results.certificate.findings.filter(f => f.risk === 'LOW').length;
		return count;
	}

	$: overallGrade = calculateOverallGrade(results);
	$: certDaysLeft = results?.certificate?.validUntil ? getDaysUntil(results.certificate.validUntil) : null;
	$: issueCount = getIssueCount(results);
	$: passedCount = getPassedCount(results);
	$: securityTxtDaysLeft = results?.securityTxt?.expiration ? getDaysUntil(results.securityTxt.expiration) : null;

	function getHealthProbeStatus(probes) {
		if (!probes?.paths) return { total: 0, found: 0 };
		const entries = Object.entries(probes.paths);
		const found = entries.filter(([_, code]) => code >= 200 && code < 400).length;
		return { total: entries.length, found };
	}

	function getExposedFilesCount(fileExposure) {
		return fileExposure?.exposedFiles?.length || 0;
	}

	function toggleSection(section) {
		expandedSections[section] = !expandedSections[section];
	}
</script>

<div class="page">
	{#if loading}
		<div class="loading-container" in:fade>
			<div class="loading-grid">
				<div class="skeleton skeleton-hero"></div>
				<div class="skeleton-sidebar">
					<div class="skeleton skeleton-title"></div>
					<div class="skeleton skeleton-grade"></div>
					<div class="skeleton skeleton-stats"></div>
				</div>
			</div>
			<div class="skeleton-findings">
				{#each Array(4) as _, i}
					<div class="skeleton skeleton-card" style="animation-delay: {i * 100}ms"></div>
				{/each}
			</div>
		</div>
	{:else if results}
		<!-- Hero Section -->
		<div class="hero" in:fade={{ duration: 300 }}>
			<!-- Screenshot -->
			<div class="screenshot-container">
				{#if !screenshotLoaded}
					<div class="screenshot-loading">
						<div class="spinner"></div>
					</div>
				{/if}
				<img
					src={`/api/screenshot/${encodeURIComponent(data.query)}`}
					alt="Website preview for {data.query}"
					class="screenshot"
					class:loaded={screenshotLoaded}
					on:load={() => screenshotLoaded = true}
					on:error={() => screenshotLoaded = true}
				/>
				<div class="screenshot-overlay">
					<a href="https://{data.query}" target="_blank" rel="noopener noreferrer" class="visit-link">
						<ExternalLink size={14} />
						<span>Visit Site</span>
					</a>
				</div>
			</div>

			<!-- Info Panel -->
			<div class="info-panel" in:fly={{ x: 20, duration: 400, delay: 100 }}>
				<div class="domain-header">
					<Globe size={20} />
					<h1>{data.query}</h1>
				</div>

				{#if results.headers?.title}
					<p class="page-title">{results.headers.title}</p>
				{/if}

				{#if results.headers?.description}
					<p class="page-description">{results.headers.description}</p>
				{/if}

				<div class="grade-display">
					<div class="grade-circle {getGradeClass(overallGrade)}">
						<span class="grade-letter">{overallGrade}</span>
					</div>
					<div class="grade-meta">
						<span class="grade-label">Overall Grade</span>
						<span class="scan-time">
							<Clock size={12} />
							{relativeTime}
						</span>
					</div>
				</div>

				<div class="quick-stats">
					<div class="stat">
						<span class="stat-value issues">{issueCount}</span>
						<span class="stat-label">Issues</span>
					</div>
					<div class="stat">
						<span class="stat-value passed">{passedCount}</span>
						<span class="stat-label">Passed</span>
					</div>
					<div class="stat">
						<span class="stat-value {certDaysLeft < 30 ? 'warning' : ''}">{certDaysLeft ?? '—'}</span>
						<span class="stat-label">Days to Cert Expiry</span>
					</div>
				</div>
			</div>
		</div>

		<!-- Score Cards -->
		<div class="score-cards" in:fade={{ duration: 300, delay: 150 }}>
			<div class="score-card">
				<div class="score-icon"><Shield size={18} /></div>
				<div class="score-content">
					<span class="score-label">Headers</span>
					<span class="score-value {getGradeClass(results.headers?.score)}">{results.headers?.score || 'N/A'}</span>
				</div>
			</div>
			<div class="score-card">
				<div class="score-icon"><Lock size={18} /></div>
				<div class="score-content">
					<span class="score-label">TLS/SSL</span>
					<span class="score-value {getGradeClass(results.certificate?.grade)}">{results.certificate?.grade || 'N/A'}</span>
				</div>
			</div>
			<div class="score-card">
				<div class="score-icon"><Server size={18} /></div>
				<div class="score-content">
					<span class="score-label">Admin</span>
					<span class="score-value risk-{results.adminPages?.risk?.toLowerCase()}">{results.adminPages?.risk || 'N/A'}</span>
				</div>
			</div>
			<div class="score-card">
				<div class="score-icon"><FileText size={18} /></div>
				<div class="score-content">
					<span class="score-label">API Docs</span>
					<span class="score-value risk-{results.swagger?.risk?.toLowerCase()}">{results.swagger?.risk || 'N/A'}</span>
				</div>
			</div>
		</div>

		<!-- Key Findings -->
		<section class="findings-section" in:fade={{ duration: 300, delay: 200 }}>
			<h2 class="section-title">
				<Activity size={18} />
				<span>Key Findings</span>
			</h2>

			<div class="findings-grid">
				<!-- Certificate Info -->
				<div class="finding-card">
					<div class="finding-header">
						<Lock size={16} />
						<span>Certificate</span>
					</div>
					<div class="finding-body">
						<div class="finding-row">
							<span class="finding-label">Issuer</span>
							<span class="finding-value">{results.certificate?.organization || 'Unknown'}</span>
						</div>
						<div class="finding-row">
							<span class="finding-label">Valid Until</span>
							<span class="finding-value" class:warning={certDaysLeft < 30}>
								{formatDate(results.certificate?.validUntil)}
								{#if certDaysLeft !== null}
									<span class="badge" class:warning={certDaysLeft < 30} class:danger={certDaysLeft < 7}>
										{certDaysLeft}d
									</span>
								{/if}
							</span>
						</div>
						<div class="finding-row">
							<span class="finding-label">TLS Version</span>
							<span class="finding-value mono">{results.certificate?.tlsVersions?.join(', ') || 'N/A'}</span>
						</div>
						<div class="finding-row">
							<span class="finding-label">Cipher</span>
							<span class="finding-value mono small">{results.certificate?.preferredCipher || 'N/A'}</span>
						</div>
					</div>
				</div>

				<!-- Infrastructure -->
				<div class="finding-card">
					<div class="finding-header">
						<Server size={16} />
						<span>Infrastructure</span>
					</div>
					<div class="finding-body">
						<div class="finding-row">
							<span class="finding-label">IP Address</span>
							<span class="finding-value mono">{results.infrastructure?.ip || 'N/A'}</span>
						</div>
						<div class="finding-row">
							<span class="finding-label">CDN</span>
							<span class="finding-value">{results.infrastructure?.cdnProvider || 'None'}</span>
						</div>
						<div class="finding-row">
							<span class="finding-label">Server</span>
							<span class="finding-value">{results.infrastructure?.server || 'Hidden'}</span>
						</div>
						<div class="finding-row">
							<span class="finding-label">HTTP</span>
							<span class="finding-value mono">{results.infrastructure?.httpVersion || 'N/A'}</span>
						</div>
					</div>
				</div>

				<!-- DNS -->
				<div class="finding-card">
					<div class="finding-header">
						<Globe size={16} />
						<span>DNS Records</span>
					</div>
					<div class="finding-body">
						<div class="finding-row">
							<span class="finding-label">A Records</span>
							<span class="finding-value mono small">{results.dnsRecords?.aRecords?.length || 0}</span>
						</div>
						<div class="finding-row">
							<span class="finding-label">NS Records</span>
							<span class="finding-value">{results.dnsRecords?.nsRecords?.length || 0}</span>
						</div>
						{#if results.dnsRecords?.findings?.length > 0}
							<div class="finding-issues">
								{#each results.dnsRecords.findings.slice(0, 3) as finding}
									<div class="issue-tag">
										<AlertTriangle size={12} />
										<span>{finding}</span>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				</div>

				<!-- Health Probes -->
				<div class="finding-card">
					<div class="finding-header">
						<Heart size={16} />
						<span>Health Endpoints</span>
					</div>
					<div class="finding-body">
						{#if results.healthProbes?.paths}
							{@const probeStatus = getHealthProbeStatus(results.healthProbes)}
							<div class="finding-row">
								<span class="finding-label">Probed</span>
								<span class="finding-value">{probeStatus.total} endpoints</span>
							</div>
							<div class="finding-row">
								<span class="finding-label">Responding</span>
								<span class="finding-value" class:success={probeStatus.found > 0}>
									{probeStatus.found}
								</span>
							</div>
							<div class="health-probes-mini">
								{#each Object.entries(results.healthProbes.paths).slice(0, 3) as [path, code]}
									<div class="probe-mini" class:probe-ok={code >= 200 && code < 400} class:probe-fail={code >= 400 || code === 0}>
										<span class="probe-code">{code || '—'}</span>
										<span class="probe-path">{path}</span>
									</div>
								{/each}
							</div>
						{:else}
							<div class="finding-row">
								<span class="finding-label">Status</span>
								<span class="finding-value">No probes</span>
							</div>
						{/if}
					</div>
				</div>
			</div>
		</section>

		<!-- Detailed Sections -->
		<div class="detail-sections" in:fade={{ duration: 300, delay: 250 }}>
			<!-- Security.txt Details -->
			<div class="detail-card">
				<button class="detail-header" on:click={() => toggleSection('securityTxt')}>
					<div class="detail-title">
						<Shield size={18} />
						<span>security.txt</span>
						{#if results.securityTxt?.exists}
							<span class="status-badge success">Found</span>
						{:else}
							<span class="status-badge warning">Missing</span>
						{/if}
					</div>
					<div class="detail-toggle">
						{#if expandedSections.securityTxt}
							<ChevronUp size={18} />
						{:else}
							<ChevronDown size={18} />
						{/if}
					</div>
				</button>
				{#if expandedSections.securityTxt}
					<div class="detail-body" transition:slide={{ duration: 200 }}>
						{#if results.securityTxt?.exists}
							<div class="detail-grid">
								{#if results.securityTxt.contacts?.length > 0}
									<div class="detail-item">
										<div class="detail-label"><Mail size={14} /> Contacts</div>
										<div class="detail-values">
											{#each results.securityTxt.contacts as contact}
												<a href={contact} class="detail-link" target="_blank" rel="noopener">
													{contact.replace('mailto:', '')}
												</a>
											{/each}
										</div>
									</div>
								{/if}
								{#if results.securityTxt.canonical?.length > 0}
									<div class="detail-item">
										<div class="detail-label"><Link size={14} /> Canonical</div>
										<div class="detail-values">
											{#each results.securityTxt.canonical as url}
												<a href={url} class="detail-link" target="_blank" rel="noopener">{url}</a>
											{/each}
										</div>
									</div>
								{/if}
								{#if results.securityTxt.encryptions?.length > 0}
									<div class="detail-item">
										<div class="detail-label"><Key size={14} /> Encryption Keys</div>
										<div class="detail-values">
											{#each results.securityTxt.encryptions as key}
												<code class="detail-code">{key}</code>
											{/each}
										</div>
									</div>
								{/if}
								{#if results.securityTxt.expiration && results.securityTxt.expiration !== '0001-01-01T00:00:00Z'}
									<div class="detail-item">
										<div class="detail-label"><Clock size={14} /> Expiration</div>
										<div class="detail-values">
											<span class:warning={securityTxtDaysLeft < 30} class:danger={securityTxtDaysLeft < 7}>
												{formatDate(results.securityTxt.expiration)}
												{#if securityTxtDaysLeft !== null}
													<span class="badge" class:warning={securityTxtDaysLeft < 30} class:danger={securityTxtDaysLeft < 7}>
														{securityTxtDaysLeft}d remaining
													</span>
												{/if}
											</span>
										</div>
									</div>
								{/if}
								<div class="detail-item">
									<div class="detail-label"><Shield size={14} /> PGP Signed</div>
									<div class="detail-values">
										{#if results.securityTxt.validSignature}
											<span class="status-inline success"><CheckCircle size={14} /> Yes</span>
										{:else}
											<span class="status-inline muted"><XCircle size={14} /> No</span>
										{/if}
									</div>
								</div>
							</div>
							{#if results.securityTxt.content}
								<div class="code-block">
									<pre>{results.securityTxt.content}</pre>
								</div>
							{/if}
						{:else}
							<div class="empty-state">
								<p>No security.txt file found at <code>/.well-known/security.txt</code></p>
								<a href="https://securitytxt.org/" target="_blank" rel="noopener" class="learn-link">
									Learn how to create one →
								</a>
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<!-- robots.txt Details -->
			<div class="detail-card">
				<button class="detail-header" on:click={() => toggleSection('robotsTxt')}>
					<div class="detail-title">
						<FileText size={18} />
						<span>robots.txt</span>
						{#if results.robotsTxt?.exists}
							<span class="status-badge success">Found</span>
						{:else}
							<span class="status-badge warning">Missing</span>
						{/if}
					</div>
					<div class="detail-toggle">
						{#if expandedSections.robotsTxt}
							<ChevronUp size={18} />
						{:else}
							<ChevronDown size={18} />
						{/if}
					</div>
				</button>
				{#if expandedSections.robotsTxt}
					<div class="detail-body" transition:slide={{ duration: 200 }}>
						{#if results.robotsTxt?.exists && results.robotsTxt?.content}
							<div class="code-block">
								<pre>{results.robotsTxt.content}</pre>
							</div>
							{#if results.robotsTxt.findings?.length > 0}
								<div class="findings-inline">
									{#each results.robotsTxt.findings as finding}
										<div class="finding-inline">
											<AlertTriangle size={14} />
											<span>{finding.description}</span>
										</div>
									{/each}
								</div>
							{/if}
						{:else}
							<div class="empty-state">
								<p>No robots.txt file found</p>
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Health Probes Details -->
			<div class="detail-card">
				<button class="detail-header" on:click={() => toggleSection('healthProbes')}>
					<div class="detail-title">
						<Heart size={18} />
						<span>Health Endpoints</span>
						<span class="status-badge {getHealthProbeStatus(results.healthProbes).found > 0 ? 'success' : 'muted'}">
							{getHealthProbeStatus(results.healthProbes).found}/{getHealthProbeStatus(results.healthProbes).total}
						</span>
					</div>
					<div class="detail-toggle">
						{#if expandedSections.healthProbes}
							<ChevronUp size={18} />
						{:else}
							<ChevronDown size={18} />
						{/if}
					</div>
				</button>
				{#if expandedSections.healthProbes}
					<div class="detail-body" transition:slide={{ duration: 200 }}>
						{#if results.healthProbes?.paths}
							<div class="probes-grid">
								{#each Object.entries(results.healthProbes.paths) as [path, code]}
									<div class="probe-item" class:probe-ok={code >= 200 && code < 400} class:probe-fail={code >= 400 || code === 0}>
										<span class="probe-status-code">{code || '—'}</span>
										<span class="probe-path-full">{path}</span>
										{#if code >= 200 && code < 400}
											<CheckCircle size={14} class="probe-icon-ok" />
										{:else}
											<XCircle size={14} class="probe-icon-fail" />
										{/if}
									</div>
								{/each}
							</div>
						{:else}
							<div class="empty-state">
								<p>No health endpoints probed</p>
							</div>
						{/if}
					</div>
				{/if}
			</div>
		</div>

		<!-- File Exposure -->
		{#if results.fileExposure?.exposedFiles?.length > 0}
			<section class="exposure-section" in:fade={{ duration: 300, delay: 300 }}>
				<h2 class="section-title danger">
					<Eye size={18} />
					<span>Exposed Files ({results.fileExposure.exposedFiles.length})</span>
				</h2>
				<div class="exposed-files">
					{#each results.fileExposure.exposedFiles as file}
						<div class="exposed-file">
							<div class="file-icon">
								<File size={16} />
							</div>
							<div class="file-info">
								<a href="https://{data.query}{file.path}" target="_blank" rel="noopener" class="file-path">
									{file.path}
								</a>
								<span class="file-desc">{file.description}</span>
							</div>
							<div class="file-risk risk-{file.risk?.toLowerCase()}">{file.risk}</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Issues List -->
		{#if results.headers?.issues?.length > 0}
			<section class="issues-section" in:fade={{ duration: 300, delay: 250 }}>
				<h2 class="section-title">
					<AlertTriangle size={18} />
					<span>Security Issues ({results.headers.issues.length})</span>
				</h2>

				<div class="issues-list">
					{#each results.headers.issues as issue, i}
						<div class="issue-item" in:fly={{ y: 10, duration: 200, delay: i * 50 }}>
							<div class="issue-risk risk-{issue.risk.toLowerCase()}">{issue.risk}</div>
							<div class="issue-content">
								<span class="issue-desc">{issue.description}</span>
								<span class="issue-evidence">{issue.evidence}</span>
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Passed Checks -->
		{#if results.headers?.passed?.length > 0}
			<section class="passed-section" in:fade={{ duration: 300, delay: 300 }}>
				<h2 class="section-title success">
					<CheckCircle size={18} />
					<span>Passed Checks ({results.headers.passed.length})</span>
				</h2>

				<div class="passed-list">
					{#each results.headers.passed as check}
						<div class="passed-item">
							<CheckCircle size={14} />
							<span>{check}</span>
						</div>
					{/each}
				</div>
			</section>
		{/if}
	{/if}
</div>

<style>
	/* Gruvbox Colors */
	:root {
		--bg0: #282828;
		--bg1: #3c3836;
		--bg2: #504945;
		--bg3: #665c54;
		--fg: #ebdbb2;
		--fg2: #d5c4a1;
		--fg3: #bdae93;
		--fg4: #a89984;
		--red: #fb4934;
		--red-dim: #cc241d;
		--green: #b8bb26;
		--green-dim: #98971a;
		--yellow: #fabd2f;
		--yellow-dim: #d79921;
		--blue: #83a598;
		--blue-dim: #458588;
		--purple: #d3869b;
		--purple-dim: #b16286;
		--aqua: #8ec07c;
		--aqua-dim: #689d6a;
		--orange: #fe8019;
		--orange-dim: #d65d0e;
	}

	.page {
		min-height: 100vh;
		padding: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}

	/* Loading States */
	.loading-container {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.loading-grid {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 1.5rem;
	}

	.skeleton {
		background: linear-gradient(90deg, var(--bg1) 25%, var(--bg2) 50%, var(--bg1) 75%);
		background-size: 200% 100%;
		animation: shimmer 1.5s infinite;
		border-radius: 0.5rem;
	}

	.skeleton-hero {
		height: 320px;
	}

	.skeleton-sidebar {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.skeleton-title {
		height: 2rem;
		width: 80%;
	}

	.skeleton-grade {
		height: 120px;
	}

	.skeleton-stats {
		height: 80px;
	}

	.skeleton-findings {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1rem;
	}

	.skeleton-card {
		height: 160px;
	}

	@keyframes shimmer {
		0% { background-position: -200% 0; }
		100% { background-position: 200% 0; }
	}

	/* Hero Section */
	.hero {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 1.5rem;
		margin-bottom: 1.5rem;
	}

	.screenshot-container {
		position: relative;
		background: var(--bg1);
		border-radius: 0.75rem;
		overflow: hidden;
		aspect-ratio: 16/9;
	}

	.screenshot-loading {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--bg1);
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 3px solid var(--bg3);
		border-top-color: var(--aqua);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.screenshot {
		width: 100%;
		height: 100%;
		object-fit: cover;
		opacity: 0;
		transition: opacity 0.3s ease;
	}

	.screenshot.loaded {
		opacity: 1;
	}

	.screenshot-overlay {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		padding: 1rem;
		background: linear-gradient(transparent, rgba(40, 40, 40, 0.95));
		display: flex;
		justify-content: flex-end;
	}

	.visit-link {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		background: var(--bg2);
		border-radius: 0.375rem;
		color: var(--fg);
		font-size: 0.8125rem;
		text-decoration: none;
		transition: background-color 0.15s ease;
	}

	.visit-link:hover {
		background: var(--bg3);
	}

	/* Info Panel */
	.info-panel {
		background: var(--bg1);
		border-radius: 0.75rem;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.domain-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		color: var(--aqua);
	}

	.domain-header h1 {
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--fg);
		word-break: break-all;
	}

	.page-title {
		font-size: 0.875rem;
		color: var(--fg4);
		margin-top: -0.5rem;
	}

	.page-description {
		font-size: 0.8125rem;
		color: var(--fg3);
		line-height: 1.5;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.grade-display {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 1rem 0;
	}

	.grade-circle {
		width: 80px;
		height: 80px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		border: 3px solid;
	}

	.grade-circle.grade-a {
		background: rgba(184, 187, 38, 0.15);
		border-color: var(--green);
	}

	.grade-circle.grade-b {
		background: rgba(131, 165, 152, 0.15);
		border-color: var(--blue);
	}

	.grade-circle.grade-c {
		background: rgba(250, 189, 47, 0.15);
		border-color: var(--yellow);
	}

	.grade-circle.grade-d {
		background: rgba(254, 128, 25, 0.15);
		border-color: var(--orange);
	}

	.grade-circle.grade-f {
		background: rgba(251, 73, 52, 0.15);
		border-color: var(--red);
	}

	.grade-letter {
		font-size: 2rem;
		font-weight: 700;
	}

	.grade-a .grade-letter { color: var(--green); }
	.grade-b .grade-letter { color: var(--blue); }
	.grade-c .grade-letter { color: var(--yellow); }
	.grade-d .grade-letter { color: var(--orange); }
	.grade-f .grade-letter { color: var(--red); }

	.grade-meta {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.grade-label {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--fg2);
	}

	.scan-time {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.75rem;
		color: var(--fg4);
	}

	.quick-stats {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.75rem;
		padding-top: 1rem;
		border-top: 1px solid var(--bg2);
	}

	.stat {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
	}

	.stat-value {
		font-size: 1.5rem;
		font-weight: 700;
		color: var(--fg);
	}

	.stat-value.issues { color: var(--orange); }
	.stat-value.passed { color: var(--green); }
	.stat-value.warning { color: var(--yellow); }

	.stat-label {
		font-size: 0.6875rem;
		color: var(--fg4);
		text-align: center;
		text-transform: uppercase;
		letter-spacing: 0.025em;
	}

	/* Score Cards */
	.score-cards {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.score-card {
		background: var(--bg1);
		border-radius: 0.5rem;
		padding: 1rem;
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.score-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2.5rem;
		height: 2.5rem;
		background: var(--bg2);
		border-radius: 0.375rem;
		color: var(--fg3);
	}

	.score-content {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.score-label {
		font-size: 0.75rem;
		color: var(--fg4);
		text-transform: uppercase;
		letter-spacing: 0.025em;
	}

	.score-value {
		font-size: 1.125rem;
		font-weight: 700;
	}

	.score-value.grade-a { color: var(--green); }
	.score-value.grade-b { color: var(--blue); }
	.score-value.grade-c { color: var(--yellow); }
	.score-value.grade-d { color: var(--orange); }
	.score-value.grade-f { color: var(--red); }
	.score-value.risk-low { color: var(--green); }
	.score-value.risk-medium { color: var(--yellow); }
	.score-value.risk-high { color: var(--orange); }
	.score-value.risk-critical { color: var(--red); }

	/* Findings Section */
	.findings-section, .issues-section, .passed-section {
		margin-bottom: 1.5rem;
	}

	.section-title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 1rem;
		font-weight: 600;
		color: var(--fg2);
		margin-bottom: 1rem;
	}

	.section-title.success {
		color: var(--green);
	}

	.findings-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1rem;
	}

	.finding-card {
		background: var(--bg1);
		border-radius: 0.5rem;
		overflow: hidden;
	}

	.finding-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.75rem 1rem;
		background: var(--bg2);
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--fg2);
	}

	.finding-body {
		padding: 0.75rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.finding-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.5rem;
	}

	.finding-label {
		font-size: 0.75rem;
		color: var(--fg4);
	}

	.finding-value {
		font-size: 0.8125rem;
		color: var(--fg);
		display: flex;
		align-items: center;
		gap: 0.375rem;
		text-align: right;
	}

	.finding-value.mono {
		font-family: ui-monospace, monospace;
		font-size: 0.75rem;
	}

	.finding-value.small {
		font-size: 0.6875rem;
	}

	.finding-value.warning {
		color: var(--yellow);
	}

	.badge {
		font-size: 0.625rem;
		padding: 0.125rem 0.375rem;
		border-radius: 0.25rem;
		background: var(--bg2);
		color: var(--fg3);
	}

	.badge.warning {
		background: rgba(250, 189, 47, 0.2);
		color: var(--yellow);
	}

	.badge.danger {
		background: rgba(251, 73, 52, 0.2);
		color: var(--red);
	}

	.finding-value.success {
		color: var(--green);
	}

	.health-probes-mini {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		margin-top: 0.5rem;
		padding-top: 0.5rem;
		border-top: 1px solid var(--bg2);
	}

	.probe-mini {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.6875rem;
	}

	.probe-code {
		font-family: ui-monospace, monospace;
		min-width: 2rem;
		text-align: center;
	}

	.probe-path {
		color: var(--fg4);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.probe-mini.probe-ok .probe-code {
		color: var(--green);
	}

	.probe-mini.probe-fail .probe-code {
		color: var(--fg4);
	}

	.finding-issues {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		margin-top: 0.5rem;
		padding-top: 0.5rem;
		border-top: 1px solid var(--bg2);
	}

	.issue-tag {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.6875rem;
		color: var(--orange);
	}

	:global(.icon-success) {
		color: var(--green);
	}

	:global(.icon-warning) {
		color: var(--fg4);
	}

	/* Issues List */
	.issues-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.issue-item {
		display: flex;
		align-items: flex-start;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		background: var(--bg1);
		border-radius: 0.375rem;
	}

	.issue-risk {
		font-size: 0.625rem;
		font-weight: 600;
		padding: 0.25rem 0.5rem;
		border-radius: 0.25rem;
		text-transform: uppercase;
		letter-spacing: 0.025em;
		flex-shrink: 0;
	}

	.issue-risk.risk-critical {
		background: rgba(251, 73, 52, 0.2);
		color: var(--red);
	}

	.issue-risk.risk-high {
		background: rgba(254, 128, 25, 0.2);
		color: var(--orange);
	}

	.issue-risk.risk-medium {
		background: rgba(250, 189, 47, 0.2);
		color: var(--yellow);
	}

	.issue-risk.risk-low {
		background: rgba(184, 187, 38, 0.2);
		color: var(--green);
	}

	.issue-content {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.issue-desc {
		font-size: 0.875rem;
		color: var(--fg);
	}

	.issue-evidence {
		font-size: 0.75rem;
		color: var(--fg4);
		font-family: ui-monospace, monospace;
	}

	/* Passed List */
	.passed-list {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.5rem;
	}

	.passed-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.75rem;
		background: var(--bg1);
		border-radius: 0.375rem;
		font-size: 0.8125rem;
		color: var(--fg3);
	}

	.passed-item :global(svg) {
		color: var(--green);
		flex-shrink: 0;
	}

	/* Detail Sections (Collapsible) */
	.detail-sections {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin-bottom: 1.5rem;
	}

	.detail-card {
		background: var(--bg1);
		border-radius: 0.5rem;
		overflow: hidden;
	}

	.detail-header {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.875rem 1rem;
		background: transparent;
		border: none;
		cursor: pointer;
		color: var(--fg2);
		transition: background-color 0.15s ease;
	}

	.detail-header:hover {
		background: var(--bg2);
	}

	.detail-title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		font-weight: 500;
	}

	.detail-toggle {
		color: var(--fg4);
	}

	.status-badge {
		font-size: 0.625rem;
		padding: 0.125rem 0.5rem;
		border-radius: 0.25rem;
		text-transform: uppercase;
		letter-spacing: 0.025em;
		font-weight: 600;
	}

	.status-badge.success {
		background: rgba(184, 187, 38, 0.2);
		color: var(--green);
	}

	.status-badge.warning {
		background: rgba(254, 128, 25, 0.2);
		color: var(--orange);
	}

	.status-badge.muted {
		background: var(--bg2);
		color: var(--fg4);
	}

	.detail-body {
		padding: 1rem;
		border-top: 1px solid var(--bg2);
	}

	.detail-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 1rem;
		margin-bottom: 1rem;
	}

	.detail-item {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.detail-label {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.75rem;
		color: var(--fg4);
		text-transform: uppercase;
		letter-spacing: 0.025em;
	}

	.detail-values {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.detail-link {
		font-size: 0.8125rem;
		color: var(--blue);
		text-decoration: none;
		word-break: break-all;
	}

	.detail-link:hover {
		text-decoration: underline;
	}

	.detail-code {
		font-family: ui-monospace, monospace;
		font-size: 0.75rem;
		color: var(--fg3);
		background: var(--bg2);
		padding: 0.25rem 0.5rem;
		border-radius: 0.25rem;
		word-break: break-all;
	}

	.status-inline {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.8125rem;
	}

	.status-inline.success {
		color: var(--green);
	}

	.status-inline.muted {
		color: var(--fg4);
	}

	.code-block {
		background: var(--bg0);
		border-radius: 0.375rem;
		padding: 1rem;
		overflow-x: auto;
	}

	.code-block pre {
		margin: 0;
		font-family: ui-monospace, monospace;
		font-size: 0.75rem;
		color: var(--fg3);
		white-space: pre-wrap;
		word-break: break-word;
	}

	.empty-state {
		text-align: center;
		padding: 1.5rem;
		color: var(--fg4);
	}

	.empty-state p {
		margin-bottom: 0.75rem;
		font-size: 0.875rem;
	}

	.empty-state code {
		font-family: ui-monospace, monospace;
		background: var(--bg2);
		padding: 0.125rem 0.375rem;
		border-radius: 0.25rem;
		font-size: 0.8125rem;
	}

	.learn-link {
		font-size: 0.8125rem;
		color: var(--blue);
		text-decoration: none;
	}

	.learn-link:hover {
		text-decoration: underline;
	}

	.findings-inline {
		margin-top: 1rem;
		padding-top: 1rem;
		border-top: 1px solid var(--bg2);
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.finding-inline {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		color: var(--orange);
	}

	/* Health Probes Grid */
	.probes-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.5rem;
	}

	.probe-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.75rem;
		background: var(--bg0);
		border-radius: 0.375rem;
		font-size: 0.8125rem;
	}

	.probe-status-code {
		font-family: ui-monospace, monospace;
		font-weight: 600;
		min-width: 2rem;
	}

	.probe-item.probe-ok .probe-status-code {
		color: var(--green);
	}

	.probe-item.probe-fail .probe-status-code {
		color: var(--fg4);
	}

	.probe-path-full {
		flex: 1;
		color: var(--fg3);
		font-family: ui-monospace, monospace;
		font-size: 0.75rem;
	}

	:global(.probe-icon-ok) {
		color: var(--green);
	}

	:global(.probe-icon-fail) {
		color: var(--fg4);
	}

	/* File Exposure */
	.exposure-section {
		margin-bottom: 1.5rem;
	}

	.section-title.danger {
		color: var(--red);
	}

	.exposed-files {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.exposed-file {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		background: var(--bg1);
		border-radius: 0.375rem;
		border-left: 3px solid var(--red);
	}

	.file-icon {
		color: var(--fg4);
	}

	.file-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.file-path {
		font-family: ui-monospace, monospace;
		font-size: 0.875rem;
		color: var(--orange);
		text-decoration: none;
	}

	.file-path:hover {
		text-decoration: underline;
	}

	.file-desc {
		font-size: 0.75rem;
		color: var(--fg4);
	}

	.file-risk {
		font-size: 0.625rem;
		font-weight: 600;
		padding: 0.25rem 0.5rem;
		border-radius: 0.25rem;
		text-transform: uppercase;
		letter-spacing: 0.025em;
	}

	.file-risk.risk-critical {
		background: rgba(251, 73, 52, 0.2);
		color: var(--red);
	}

	.file-risk.risk-high {
		background: rgba(254, 128, 25, 0.2);
		color: var(--orange);
	}

	.file-risk.risk-medium {
		background: rgba(250, 189, 47, 0.2);
		color: var(--yellow);
	}

	.file-risk.risk-low {
		background: rgba(184, 187, 38, 0.2);
		color: var(--green);
	}

	.warning {
		color: var(--yellow);
	}

	.danger {
		color: var(--red);
	}

	/* Responsive */
	@media (max-width: 1200px) {
		.findings-grid {
			grid-template-columns: repeat(2, 1fr);
		}

		.score-cards {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	@media (max-width: 768px) {
		.hero {
			grid-template-columns: 1fr;
		}

		.loading-grid {
			grid-template-columns: 1fr;
		}

		.skeleton-findings {
			grid-template-columns: repeat(2, 1fr);
		}

		.findings-grid {
			grid-template-columns: 1fr;
		}

		.passed-list {
			grid-template-columns: 1fr;
		}

		.detail-grid {
			grid-template-columns: 1fr;
		}

		.probes-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
