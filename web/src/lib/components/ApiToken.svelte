<script>
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { Copy, Eye, EyeOff } from 'lucide-svelte';

	let apiToken = '';
	let copied = false;
	let visible = false;

	onMount(async () => {
		const response = await fetch('/api/api-token');
		if (response.ok) {
			const data = await response.json();
			apiToken = data.api_token;
		}
	});

	function copyToClipboard() {
		navigator.clipboard.writeText(apiToken);
		copied = true;
		setTimeout(() => {
			copied = false;
		}, 2000);
	}

	function toggleVisibility() {
		visible = !visible;
	}
</script>

<div class="mb-6 mt-6">
	<h3 class="text-lg font-semibold mb-2">API Token</h3>
	<p class="text-gray-400 text-sm mb-3">Use this token to authenticate API requests</p>

	<div class="relative">
		<input
			type={visible ? 'text' : 'password'}
			value={visible ? apiToken : '••••••••••••••••••••••'}
			readonly
			class="w-full px-4 py-2 text-foreground bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 pr-20"
		/>

		<div class="absolute right-2 top-1/2 transform -translate-y-1/2 flex gap-1">
			<button
				on:click={toggleVisibility}
				class="p-1.5 text-gray-400 hover:text-white transition-colors rounded-md"
				title={visible ? 'Hide token' : 'Show token'}
			>
				{#if visible}
					<EyeOff size={18} />
				{:else}
					<Eye size={18} />
				{/if}
			</button>

			<button
				on:click={copyToClipboard}
				class="p-1.5 text-gray-400 hover:text-white transition-colors rounded-md"
				title="Copy to clipboard"
			>
				<Copy size={18} />
			</button>
		</div>
	</div>

	{#if copied}
		<p class="mt-2 text-sm text-green-500" transition:fade>Copied to clipboard!</p>
	{/if}
</div>
