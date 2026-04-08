<script>
	import { CheckCircle } from 'lucide-svelte';

	let { checks = [] } = $props();

	// Normalize checks - handle string arrays, and objects with "name" or "path"
	let passedChecks = $derived((checks || [])
		.map((check) => {
			if (typeof check === 'string') {
				return { name: check, passed: true };
			}
			return { ...check, name: check.name || check.path };
		})
		.filter((check) => check.passed));
</script>

{#if passedChecks.length > 0}
	<div class="checks-grid">
		{#each passedChecks as check}
			<div class="check-item">
				<CheckCircle size={14} class="check-icon" />
				<span class="check-name">{check.name}</span>
			</div>
		{/each}
	</div>
{/if}

<style>
	.checks-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, max-content));
		gap: 0.25rem 2rem;
		margin-top: 1rem;
		padding-top: 1rem;
		border-top: 1px solid #2b2b2b;
		overflow: hidden;
	}

	@media (min-width: 640px) {
		.checks-grid {
			grid-template-columns: repeat(4, minmax(0, max-content));
		}
	}

	@media (min-width: 900px) {
		.checks-grid {
			grid-template-columns: repeat(5, minmax(0, max-content));
		}
	}

	@media (min-width: 1200px) {
		.checks-grid {
			grid-template-columns: repeat(6, minmax(0, max-content));
		}
	}

	.check-item {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0;
		font-size: 0.8125rem;
		min-width: 0;
	}

	:global(.check-icon) {
		color: #22c55e;
		flex-shrink: 0;
	}

	.check-name {
		color: #9ca3af;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
