<script>
	import '../app.css';
	/**
	 * @typedef {Object} Props
	 * @property {import('svelte').Snippet} [children]
	 */

	/** @type {Props} */
	let { children } = $props();
</script>

<div
	class="min-h-screen w-full bg-background text-foreground flex flex-col sm:flex-row overflow-hidden"
>
	{@render children?.()}
</div>

<style global>
	:global(body),
	:global(html) {
		background: #222;
		color: #fff;
		width: 100%;
		overflow-x: hidden;
	}

	:global(.tooltip) {
		position: relative;
		display: inline-block;
	}

	:global(.tooltip:hover::after) {
		content: attr(data-tooltip);
		position: fixed;
		transform: translateX(-50%);
		padding: 0.5rem;
		background: #222;
		color: white;
		border-radius: 0.25rem;
		font-size: 0.7em;
		white-space: nowrap;
		z-index: 50;
		left: var(--tooltip-x);
		top: var(--tooltip-y);
	}
</style>
