<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { slide, fade } from 'svelte/transition';
  import { clickOutside } from '$lib/actions/clickOutside';

  export let value = '';
  export let options = [];
  export let placeholder = 'Select...';
  export let icon = null;

  const dispatch = createEventDispatcher();

  let isOpen = false;
  let selectedOption = options.find(option => option.value === value);
  let selectContainer;

  function toggle() {
    isOpen = !isOpen;
  }

  function handleSelect(option) {
    selectedOption = option;
    value = option.value;
    dispatch('change', { value: option.value });
    isOpen = false;
  }

  function handleClickOutside() {
    isOpen = false;
  }

  $: if (value) {
    selectedOption = options.find(option => option.value === value) || null;
  }

  onMount(() => {
    if (value && !selectedOption) {
      selectedOption = options.find(option => option.value === value) || null;
    }
  });
</script>

<div
  class="custom-select relative w-[170px]"
  use:clickOutside={{ enabled: isOpen, cb: handleClickOutside }}
  bind:this={selectContainer}
>
  <button
    type="button"
    class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 flex items-center w-full text-left transition-colors disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-green-500/70"
    on:click={toggle}
    aria-haspopup="listbox"
    aria-expanded={isOpen}
    data-testid="custom-select-button"
  >
    <span class="flex-1">
      {#if selectedOption && selectedOption.icon && typeof selectedOption.icon === 'string'}
        <span class="flex items-center">{@html selectedOption.icon}</span>
      {:else}
        {selectedOption ? selectedOption.label : placeholder}
      {/if}
    </span>
    <span class="ml-2 select-arrow">
      <svg
        class="h-5 w-5 text-green-500 transition-transform duration-300 {isOpen ? 'rotate-180' : ''}"
        fill="currentColor"
        viewBox="0 0 20 20"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
      </svg>
    </span>
  </button>

  {#if isOpen}
    <div
      class="absolute z-10 w-full mt-1.5 bg-[#2b2b2b] rounded-lg overflow-hidden max-h-64 overflow-y-auto"
      in:slide={{ duration: 200, easing: t => t * (2 - t) }}
      out:slide={{ duration: 150 }}
      role="listbox"
      data-testid="custom-select-options"
    >
      {#each options as option}
        <div
          class="py-2.5 px-4 cursor-pointer hover:bg-green-900/30 transition-colors flex items-center relative group {option.value === value ? 'bg-green-900/30 text-green-300 font-medium' : 'text-gray-200'}"
          role="option"
          aria-selected={option.value === value}
          on:click={() => handleSelect(option)}
          in:fade={{ duration: 100, delay: 50 * options.indexOf(option) }}
        >
          <div class="absolute left-0 top-0 bottom-0 w-0.5 bg-green-500/0 group-hover:bg-green-500/70 transition-all {option.value === value ? 'bg-green-500/70' : ''}"></div>
          {#if option.icon}
            <span class="mr-2">{@html option.icon}</span>
          {:else}
            <span class="truncate">{option.label}</span>
          {/if}
          {#if option.value === value}
            <span class="ml-auto text-green-400">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"></polyline>
              </svg>
            </span>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  /* Button styling is now handled via Tailwind classes */

  .custom-select button:focus {
    animation: pulse 1.5s ease-in-out infinite alternate;
  }

  /* Dropdown container styling */
  .custom-select [role="listbox"] {
    /* box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3), 0 0 8px rgba(34, 197, 94, 0.2); */
    /* background-image: linear-gradient(to bottom, rgba(34, 197, 94, 0.05), transparent); */
    backdrop-filter: blur(12px);
  }

  /* Dropdown option hover effect */
  .custom-select [role="option"]:hover {
    background-color: rgba(34, 197, 94, 0.15);
  }

  /* Select arrow animation */
  .select-arrow svg {
    filter: drop-shadow(0 0 2px rgba(34, 197, 94, 0.3));
  }

  /* Selected option styling */
  .custom-select [role="option"][aria-selected="true"] {
    background-color: rgba(34, 197, 94, 0.2);
  }

  @keyframes pulse {
    from {
      box-shadow: 0 0 0 rgba(34, 197, 94, 0.4);
    }
    to {
      box-shadow: 0 0 10px rgba(34, 197, 94, 0.7);
    }
  }

  /* Custom scrollbar for the dropdown */
  .custom-select [role="listbox"]::-webkit-scrollbar {
    width: 6px;
  }

  .custom-select [role="listbox"]::-webkit-scrollbar-track {
    background: rgba(0, 0, 0, 0.2);
    border-radius: 10px;
  }

  .custom-select [role="listbox"]::-webkit-scrollbar-thumb {
    background: rgba(34, 197, 94, 0.5);
    border-radius: 10px;
  }

  .custom-select [role="listbox"]::-webkit-scrollbar-thumb:hover {
    background: rgba(34, 197, 94, 0.7);
  }
</style>
