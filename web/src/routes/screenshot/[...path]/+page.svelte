<script>
	
	/**
	 * @typedef {Object} Props
	 * @property {import('./$types').PageData} data
	 */

	/** @type {Props} */
	let { data } = $props();

	let loading = $state(true);
	let error = $state('');

	function handleImageLoad() {
		loading = false;
	}

	function handleImageError() {
		loading = false;
		error = 'Failed to load screenshot';
	}

</script>

<svelte:head>
	<title>Screenshot - {data.domain}</title>
</svelte:head>

<div class="screenshot-container">
	{#if loading}
		<div class="loading">Loading screenshot...</div>
	{:else if error}
		<div class="error">{error}</div>
	{/if}
	<img 
		src={data.imageSrc} 
		alt="Screenshot of {data.domain}"
		class:hidden={loading || error}
		onload={handleImageLoad}
		onerror={handleImageError}
	/>
</div>

<style>
	.screenshot-container {
		width: 100%;
		height: 100vh;
		display: flex;
		justify-content: center;
		align-items: center;
		background: #1a1a1a;
	}

	img {
		max-width: 100%;
		max-height: 100vh;
		object-fit: contain;
	}

	img.hidden {
		display: none;
	}

	.loading, .error {
		color: #fff;
		font-size: 1.2rem;
		padding: 2rem;
	}

	.error {
		color: #ff4444;
	}
</style>
