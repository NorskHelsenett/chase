<!-- ToggleButton.svelte -->
<script lang="ts">
	interface Props {
		value?: boolean;
		onLabel?: string;
		offLabel?: string;
		activeColor?: string;
		disabled?: boolean;
		size?: 'sm' | 'md' | 'lg';
		id?: string;
		onchange?: (value: boolean) => void;
	}

	let {
		value = $bindable(true),
		onLabel = 'Active',
		offLabel = 'Inactive',
		activeColor = 'bg-green-600',
		disabled = false,
		size = 'md',
		id = '',
		onchange
	}: Props = $props();

	// Calculate size classes
	const sizeClasses = {
		sm: 'px-2 py-1 text-xs',
		md: 'px-3 py-1.5 text-sm',
		lg: 'px-4 py-2 text-base'
	};

	function setValue(val: boolean) {
		if (!disabled) {
			value = val;
			onchange?.(value);
		}
	}

	// Handle keyboard navigation
	function handleKeydown(event: KeyboardEvent, val: boolean) {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			setValue(val);
		}
	}
</script>

<div
	class="flex bg-[#2b2b2b] rounded-lg p-0.5 w-[145px] p-1 relative overflow-hidden"
	role="group"
	{id}
>
	<!-- Animated background highlight -->
	<div
		class="absolute top-0 left-0 h-full rounded transform transition-transform duration-300 ease-in-out {activeColor}"
		style="width: 50%; transform: translateX({value ? '0%' : '100%'});"
	></div>

	<button
		type="button"
		class="{sizeClasses[
			size
		]} rounded transition-all duration-300 ease-in-out focus:outline-none relative z-10 flex-1 {value
			? 'text-white'
			: 'text-gray-400 hover:text-gray-200'}"
		onclick={() => setValue(!value)}
		onkeydown={(e) => handleKeydown(e, true)}
		{disabled}
		aria-pressed={value}
		tabindex={disabled ? -1 : 0}
	>
		{onLabel}
	</button>
	<button
		type="button"
		class="{sizeClasses[
			size
		]} rounded transition-all duration-300 ease-in-out focus:outline-none relative z-10 flex-1 {!value
			? 'text-white'
			: 'text-gray-400 hover:text-gray-200'}"
		onclick={() => setValue(!value)}
		onkeydown={(e) => handleKeydown(e, false)}
		{disabled}
		aria-pressed={!value}
		tabindex={disabled ? -1 : 0}
	>
		{offLabel}
	</button>
</div>
