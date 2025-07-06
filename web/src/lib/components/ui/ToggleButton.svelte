<!-- ToggleButton.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let value: boolean = true;
  export let onLabel: string = 'Active';
  export let offLabel: string = 'Inactive';
  export let activeColor: string = 'bg-green-600';
  export let disabled: boolean = false;
  export let size: 'sm' | 'md' | 'lg' = 'md';
  export let id: string = '';

  // Calculate size classes
  const sizeClasses = {
    sm: 'px-2 py-1 text-xs',
    md: 'px-3 py-1.5 text-sm',
    lg: 'px-4 py-2 text-base'
  };

  const dispatch = createEventDispatcher();

  function setValue(val: boolean) {
    if (!disabled) {
      value = val;
      dispatch('change', value);
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

<div class="flex bg-[#2b2b2b] rounded-lg p-0.5 w-[145px] p-1 relative overflow-hidden" role="group" {id}>
  <!-- Animated background highlight -->
  <div 
    class="absolute top-0 left-0 h-full rounded transform transition-transform duration-300 ease-in-out {activeColor}" 
    style="width: 50%; transform: translateX({value ? '0%' : '100%'});"
  ></div>
  
  <button
    type="button"
    class="{sizeClasses[size]} rounded transition-all duration-300 ease-in-out focus:outline-none relative z-10 flex-1 {value ? 'text-white' : 'text-gray-400 hover:text-gray-200'}"
    on:click={() => setValue(!value)}
    on:keydown={(e) => handleKeydown(e, true)}
    disabled={disabled}
    aria-pressed={value}
    tabindex={disabled ? -1 : 0}
  >
    {onLabel}
  </button>
  <button
    type="button"
    class="{sizeClasses[size]} rounded transition-all duration-300 ease-in-out focus:outline-none relative z-10 flex-1 {!value ? 'text-white' : 'text-gray-400 hover:text-gray-200'}"
    on:click={() => setValue(!value)}
    on:keydown={(e) => handleKeydown(e, false)}
    disabled={disabled}
    aria-pressed={!value}
    tabindex={disabled ? -1 : 0}
  >
    {offLabel}
  </button>
</div>
