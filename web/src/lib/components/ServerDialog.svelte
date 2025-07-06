<!-- ServerDialog.svelte -->
<script lang="ts">
	import { fade } from 'svelte/transition';
	import { createEventDispatcher } from 'svelte';
	import type { Server } from '$lib/models';
	import CustomCheckbox from './ui/CustomCheckbox.svelte';
	import IntervalSlider from './ui/IntervalSlider.svelte';
	import RadioToggle from './ui/RadioToggle.svelte';

	const dispatch = createEventDispatcher();

	export let showDialog = false;
	export let isLoading = false;
	export let initialData: Partial<Server> | null = null;
	export let mode: 'add' | 'edit' = 'add';

	// Default values
	const defaultFormData = {
		id: undefined as number | undefined,
		url: '',
		active: true,
		follow_redirect: true,
		allow_insecure: false,
		expected_status: 200,
		comment: '',
		update_interval: 15
	};

	// Reactive values
	let formData = { ...defaultFormData };
	let expectedDown = false;
	let currentStatus = true;
	let intervalValue = 15;
	let hasInitialized = false;

	// UI text based on mode
	$: title = mode === 'add' ? 'Add New Server' : 'Edit Server';
	$: submitLabel = mode === 'add' ? 'Add Server' : 'Save Changes';
	$: loadingLabel = mode === 'add' ? 'Adding...' : 'Saving...';

	// Reset all form values to defaults
	function resetForm() {
		formData = { ...defaultFormData };
		expectedDown = false;
		currentStatus = defaultFormData.active;
		intervalValue = defaultFormData.update_interval;
		hasInitialized = false;
	}

	// Initialize form when dialog opens
	$: if (showDialog && !hasInitialized) {
		hasInitialized = true;

		if (mode === 'add') {
			resetForm();
		} else if (initialData) {
			// Map initialData to form values, using default values as fallbacks
			formData = {
				id: initialData.id,
				url: initialData.url || defaultFormData.url,
				active: initialData.active ?? defaultFormData.active,
				follow_redirect: initialData.follow_redirect ?? defaultFormData.follow_redirect,
				allow_insecure: initialData.allow_insecure ?? defaultFormData.allow_insecure,
				expected_status: initialData.expected_status ?? defaultFormData.expected_status,
				comment: initialData.comment || defaultFormData.comment,
				update_interval: initialData.update_interval ?? defaultFormData.update_interval
			};

			// Derive other values from initialData
			expectedDown = initialData.expected_status === 0;
			currentStatus = initialData.active ?? defaultFormData.active;
			intervalValue = initialData.update_interval ?? defaultFormData.update_interval;
		}
	}

	// Reset initialization flag when dialog closes
	$: if (!showDialog) {
		hasInitialized = false;
	}

	function handleSubmit() {
		const serverData = {
			...formData,
			expected_status: expectedDown ? 0 : formData.expected_status,
			active: currentStatus,
			update_interval: intervalValue
		};

		dispatch('submit', {
			data: serverData,
			mode
		});
	}

	function handleClose() {
		showDialog = false;
		dispatch('close');
	}

	function handleIntervalChange(event: CustomEvent) {
		intervalValue = event.detail;
	}

	function handleStatusChange(event: CustomEvent) {
		currentStatus = event.detail;
	}

	function handleCheckboxChange(field: string, event: CustomEvent) {
		formData[field] = event.detail;
	}
</script>

{#if showDialog}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" transition:fade>
		<div class="bg-[#202020] rounded-lg p-6 w-full max-w-xl">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-xl text-gray-200 font-semibold">{title}</h2>
				{#if mode === 'edit'}
					<RadioToggle value={currentStatus} on:change={handleStatusChange} label="Status" />
				{/if}
			</div>

			<form on:submit|preventDefault={handleSubmit} class="space-y-4">
				<div>
					<label class="block text-gray-300 mb-1" for="url">URL</label>
					<input
						id="url"
						type="text"
						bind:value={formData.url}
						required
						autofocus
						class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-green-500"
						placeholder="https://example.com"
					/>
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div class="space-y-4">
						<CustomCheckbox
							checked={formData.follow_redirect}
							on:change={(e) => handleCheckboxChange('follow_redirect', e)}
							label="Follow Redirects"
						/>

						<CustomCheckbox
							checked={formData.allow_insecure}
							on:change={(e) => handleCheckboxChange('allow_insecure', e)}
							label="Allow Insecure"
						/>
					</div>

					<div class="space-y-4">
						<CustomCheckbox
							bind:checked={expectedDown}
							on:change={(e) => (expectedDown = e.detail)}
							label="Expected Down"
						/>
					</div>
				</div>

				{#if !expectedDown}
					<div>
						<label class="block text-gray-300 mb-1" for="status">Expected Status</label>
						<input
							id="status"
							type="number"
							bind:value={formData.expected_status}
							min="100"
							max="599"
							class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-green-500"
						/>
					</div>
				{/if}

				<div>
					<IntervalSlider
						value={intervalValue}
						on:change={handleIntervalChange}
						label="Check Interval"
					/>
				</div>

				<div>
					<label class="block text-gray-300 mb-1" for="comment">Note</label>
					<textarea
						id="comment"
						bind:value={formData.comment}
						rows="3"
						class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-green-500"
						placeholder="Add any notes about this server..."
					></textarea>
				</div>

				<div class="flex justify-end gap-3 mt-6">
					<button
						type="button"
						on:click={handleClose}
						disabled={isLoading}
						class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={isLoading}
						class="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
					>
						{#if isLoading}
							<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
									fill="none"
								/>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								/>
							</svg>
						{/if}
						{isLoading ? loadingLabel : submitLabel}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
