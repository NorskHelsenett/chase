<script lang="ts">
  export let checked = false;
  export let label = '';
  
  function handleClick() {
    checked = !checked;
  }
</script>

<label 
  class="flex items-center gap-2.5 cursor-pointer select-none group"
  on:click={handleClick}
>
  <div 
    class="w-5 h-5 rounded flex items-center justify-center transition-all duration-200 
    {checked ? 'bg-green-600 border-green-600' : 'bg-[#2b2b2b] hover:bg-[#333]'}"
  >
    {#if checked}
      <svg 
        class="w-3.5 h-3.5 text-white" 
        fill="none" 
        stroke="currentColor" 
        viewBox="0 0 24 24"
      >
        <path 
          stroke-linecap="round" 
          stroke-linejoin="round" 
          stroke-width="2.5" 
          d="M5 13l4 4L19 7"
        />
      </svg>
    {/if}
  </div>
  <span class="text-gray-300 text-sm group-hover:text-gray-200 transition-colors">
    {label}
  </span>
</label>