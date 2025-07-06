<!-- RadioToggle.svelte -->
<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	const dispatch = createEventDispatcher();

	export let value = true;
	export let label = '';

	const options = [
		{ value: true, label: 'Active' },
		{ value: false, label: 'Inactive' }
	];

	function handleClick(newValue: boolean) {
		dispatch('change', newValue);
	}
</script>

<div class="flex items-center gap-4">
	{#if label}
		<span class="text-gray-300 text-sm">{label}</span>
	{/if}
	<div class="flex bg-[#2b2b2b] rounded-lg p-0.5">
		{#each options as option}
			<button
				type="button"
				class="px-3 py-1.5 rounded text-sm transition-all duration-200 {value === option.value
					? 'bg-green-600 text-white'
					: 'text-gray-400 hover:text-gray-200'}"
				on:click={() => handleClick(option.value)}
			>
				{option.label}
			</button>
		{/each}
	</div>
</div>
