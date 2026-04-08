<script lang="ts">
	import { CircleCheck, CircleX, KeyRound, ShieldAlert } from 'lucide-svelte';

	interface Props {
		stats?: {
		up: number;
		down: number;
		secretsExposed: number;
		highRisks: number;
	} | null;
	}

	let { stats = null }: Props = $props();

	let total = $derived(stats ? stats.up + stats.down : 0);
</script>

<div class="stats-row">
	<div class="stat-card">
		<div class="stat-icon icon-green">
			<CircleCheck size={24} />
		</div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{stats.up}<span class="stat-sub">/{total}</span></span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Online</span>
		</div>
	</div>

	<div class="stat-card">
		<div class="stat-icon icon-red">
			<CircleX size={24} />
		</div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{stats.down}</span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Offline</span>
		</div>
	</div>

	<div class="stat-card">
		<div class="stat-icon icon-purple">
			<KeyRound size={24} />
		</div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{stats.secretsExposed}</span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Secrets Exposed</span>
		</div>
	</div>

	<div class="stat-card">
		<div class="stat-icon icon-orange">
			<ShieldAlert size={24} />
		</div>
		<div class="stat-body">
			{#if stats}
				<span class="stat-number">{stats.highRisks}</span>
			{:else}
				<span class="stat-skeleton"></span>
			{/if}
			<span class="stat-label">Security Risks</span>
		</div>
	</div>
</div>

<style>
	.stats-row {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
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
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	@media (max-width: 768px) {
		.stats-row {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	@media (max-width: 420px) {
		.stats-row {
			grid-template-columns: 1fr;
		}
	}
</style>
