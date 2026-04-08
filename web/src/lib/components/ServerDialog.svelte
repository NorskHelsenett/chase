<!-- ServerDialog.svelte -->
<script lang="ts">
	import { run, preventDefault } from 'svelte/legacy';

	import { fade } from 'svelte/transition';
	import type { Server } from '$lib/models';
	import CustomCheckbox from './ui/CustomCheckbox.svelte';
	import IntervalSlider from './ui/IntervalSlider.svelte';
	import RadioToggle from './ui/RadioToggle.svelte';

	interface Props {
		showDialog?: boolean;
		isLoading?: boolean;
		initialData?: Partial<Server> | null;
		mode?: 'add' | 'edit';
		onsubmit?: (detail: { data: any; mode: string }) => void;
		onclose?: () => void;
		ontoggleActive?: (value: boolean) => void;
		ondelete?: () => void;
	}

	let {
		showDialog = $bindable(false),
		isLoading = false,
		initialData = null,
		mode = 'add',
		onsubmit,
		onclose,
		ontoggleActive,
		ondelete
	}: Props = $props();

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
	let formData = $state({ ...defaultFormData });
	let expectedDown = $state(false);
	let currentStatus = $state(true);
	let intervalValue = $state(15);
	let hasInitialized = $state(false);

	// UI text based on mode
	let title = $derived(mode === 'add' ? 'Add New Server' : 'Edit Server');
	let submitLabel = $derived(mode === 'add' ? 'Add Server' : 'Save Changes');
	let loadingLabel = $derived(mode === 'add' ? 'Adding...' : 'Saving...');

	// Reset all form values to defaults
	function resetForm() {
		formData = { ...defaultFormData };
		expectedDown = false;
		currentStatus = defaultFormData.active;
		intervalValue = defaultFormData.update_interval;
		hasInitialized = false;
	}

	// Initialize form when dialog opens
	run(() => {
		if (showDialog && !hasInitialized) {
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
	});

	// Reset initialization flag when dialog closes
	run(() => {
		if (!showDialog) {
			hasInitialized = false;
		}
	});

	function handleSubmit() {
		const serverData = {
			...formData,
			expected_status: expectedDown ? 0 : formData.expected_status,
			active: currentStatus,
			update_interval: intervalValue
		};

		onsubmit?.({
			data: serverData,
			mode
		});
	}

	function handleClose() {
		showDialog = false;
		onclose?.();
	}

	function handleIntervalChange(value: number) {
		intervalValue = value;
	}

	function handleStatusChange(value: boolean) {
		currentStatus = value;
		// Dispatch toggle event for immediate update
		ontoggleActive?.(value);
	}

	function handleDelete() {
		ondelete?.();
	}

	function handleCheckboxChange(field: string, value: boolean) {
		formData[field] = value;
	}
</script>

{#if showDialog}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" transition:fade>
		<div class="bg-[#202020] rounded-lg p-6 w-full max-w-xl">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-xl text-gray-200 font-semibold">{title}</h2>
				{#if mode === 'edit'}
					<RadioToggle value={currentStatus} onchange={handleStatusChange} label="Status" />
				{/if}
			</div>

			<form onsubmit={preventDefault(handleSubmit)} class="space-y-4">
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
							onchange={(value) => handleCheckboxChange('follow_redirect', value)}
							label="Follow Redirects"
						/>

						<CustomCheckbox
							checked={formData.allow_insecure}
							onchange={(value) => handleCheckboxChange('allow_insecure', value)}
							label="Allow Insecure"
						/>
					</div>

					<div class="space-y-4">
						<CustomCheckbox
							bind:checked={expectedDown}
							onchange={(value) => (expectedDown = value)}
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
						onchange={handleIntervalChange}
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

				<div class="flex justify-between mt-6">
					{#if mode === 'edit'}
						<button
							type="button"
							onclick={handleDelete}
							disabled={isLoading}
							class="px-4 py-2 bg-red-600/10 hover:bg-red-600/20 rounded-lg text-red-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
								/>
							</svg>
							Delete
						</button>
					{:else}
						<div></div>
					{/if}
					<div class="flex gap-3">
						<button
							type="button"
							onclick={handleClose}
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
				</div>
			</form>
		</div>
	</div>
{/if}
