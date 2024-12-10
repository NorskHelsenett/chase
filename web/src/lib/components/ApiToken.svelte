<script>
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  let apiToken = '';
  let copied = false;

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
</script>

<div class="relative">
  <input
      type="text"
      value={apiToken}
      disabled
      class="w-full px-4 py-2 text-foreground bg-background border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
  />
  <button
      on:click={copyToClipboard}
      class="absolute right-2 top-1/2 transform -translate-y-1/2 px-3 py-1 text-sm text-primary-foreground bg-primary rounded-md hover:bg-opacity-90 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 transition-colors duration-300"
  >
      Copy
  </button>
</div>
{#if copied}
  <p class="mt-2 text-sm text-primary" transition:fade>Copied to clipboard!</p>
{/if}