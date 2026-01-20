<script lang="ts">
	export let stats: {
		up: number;
		down: number;
		criticalRisks: number;
		highRisks: number;
	} | null = null;
</script>

<div class="stats-card">
	<div class="stats-grid">
		<div class="stat-item">
			<div class="stat-label">Online</div>
			{#if stats}
				<div class="stat-value success">{stats.up}</div>
			{:else}
				<div class="stat-skeleton"></div>
			{/if}
		</div>

		<div class="stat-item">
			<div class="stat-label">Offline</div>
			{#if stats}
				<div class="stat-value error">{stats.down}</div>
			{:else}
				<div class="stat-skeleton"></div>
			{/if}
		</div>

		<div class="stat-item">
			<div class="stat-label">Critical Risks</div>
			{#if stats}
				<div class="stat-value error">{stats.criticalRisks}</div>
			{:else}
				<div class="stat-skeleton"></div>
			{/if}
		</div>

		<div class="stat-item">
			<div class="stat-label">High Risks</div>
			{#if stats}
				<div class="stat-value warning">{stats.highRisks}</div>
			{:else}
				<div class="stat-skeleton"></div>
			{/if}
		</div>
	</div>
</div>

<style>
	.stats-card {
		background: #202020;
		border-radius: 0.5rem;
		padding: 1rem;
		margin-bottom: 1rem;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 2rem;
	}

	.stat-item {
		text-align: center;
	}

	.stat-label {
		font-size: 0.875rem;
		font-weight: 500;
		color: #9ca3af;
		margin-bottom: 0.5rem;
	}

	.stat-value {
		font-size: 1.5rem;
		font-weight: 600;
	}

	.stat-value.success {
		color: #22c55e;
	}

	.stat-value.error {
		color: #ef4444;
	}

	.stat-value.warning {
		color: #f97316;
	}

	.stat-skeleton {
		height: 2rem;
		width: 3rem;
		margin: 0 auto;
		background: #374151;
		border-radius: 0.25rem;
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	@media (max-width: 640px) {
		.stats-grid {
			grid-template-columns: repeat(2, 1fr);
			gap: 1rem;
		}
	}
</style>
