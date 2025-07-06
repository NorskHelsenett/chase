<script>
	import { fade, fly } from 'svelte/transition';
	import { goto } from '$app/navigation';

	let searchText = '';
	let isAnimating = false;

	function handleSearch() {
		if (!searchText.trim() || isAnimating) return;

		isAnimating = true;
		setTimeout(() => {
			goto(`/search/${encodeURIComponent(searchText.trim())}`);
		}, 100);
	}

	function handleKeydown(event) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			handleSearch();
		}
	}
</script>

<div class="max-h-screen min-h-[90vh] flex items-center justify-center font-system">
	<div class="w-full max-w-[768px] p-4" in:fade={{ duration: 300 }}>
		<h1 class="text-white text-5xl text-center mb-8" in:fly={{ y: -20, duration: 500, delay: 150 }}>
			Explore a domain
		</h1>

		<div
			class="bg-[#202020]/50 rounded-lg p-4 border-2 border-[#3e3e3e] transition-all duration-500 ease-in-out {isAnimating
				? 'scale-98 opacity-0'
				: ''}"
			in:fly={{ y: 20, duration: 500, delay: 300 }}
		>
			<textarea
				bind:value={searchText}
				on:keydown={handleKeydown}
				placeholder="Any domain..."
				rows="1"
				class="w-[25em] bg-transparent border-none text-[#bfbfbf] text-lg outline-none resize-none placeholder-[#918c8c]"
			/>
			<div class="flex justify-between items-center mt-4 text-[#bfbfbf]">
				<div class="flex gap-4 items-center">
					<!-- <span>Focus</span>
                    <span>Attach</span> -->
				</div>
				<div class="flex gap-4 items-center">
					<button
						on:click={handleSearch}
						disabled={isAnimating}
						class="h-10 w-10 text-gray-300 hover:bg-gray-600 disabled:opacity-50 disabled:hover:bg-gray-700 transition-colors duration-200 ease-in-out p-2 text-white bg-gray-700 rounded-full"
					>
						→
					</button>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	/* Custom font-family and scale utility */
	.font-system {
		font-family:
			system-ui,
			-apple-system,
			sans-serif;
	}
	.scale-98 {
		transform: scale(0.98);
	}
</style>
