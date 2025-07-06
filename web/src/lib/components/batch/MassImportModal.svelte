<!-- MassImportModal.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import CustomCheckbox from '../ui/CustomCheckbox.svelte';
  import IntervalSlider from '../ui/IntervalSlider.svelte';

  const dispatch = createEventDispatcher();

  export let showModal = false;
  export let isLoading = false;

  // Default settings
  let intervalValue = 15; // Default check interval (15 min)
  let followRedirect = true;
  let allowInsecure = false;
  let sites = '';
  const placeholderText = `https://example.com
https://example.org
https://another-site.com`;
  
  // Process input to handle different separators (newlines, commas, semicolons)
  function processSiteInput(input: string): string[] {
    if (!input.trim()) return [];
    
    // First split by newlines
    const lines = input.split(/\n+/);
    const result: string[] = [];
    
    // Process each line for comma or semicolon separated values
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      
      // Check if line contains commas or semicolons
      if (trimmed.includes(',') || trimmed.includes(';')) {
        // Split by both comma and semicolon
        const entries = trimmed.split(/[,;]+/);
        for (const entry of entries) {
          const cleanEntry = entry.trim();
          if (cleanEntry) result.push(cleanEntry);
        }
      } else {
        result.push(trimmed);
      }
    }
    
    return result;
  }
  
  // Submit form data
  async function handleSubmit() {
    isLoading = true;

    try {
      // Process sites using our helper function that handles multiple separators
      const sitesList = processSiteInput(sites);

      if (sitesList.length === 0) {
        alert('Please enter at least one site to import');
        isLoading = false;
        return;
      }

      // Prepare data for submission with global settings
      const importData = {
        sites: sitesList,
        settings: {
          update_interval: intervalValue,
          follow_redirect: followRedirect,
          allow_insecure: allowInsecure,
        }
      };

      // Send to API
      const response = await fetch('/api/servers/batch-import', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(importData),
      });

      if (!response.ok) {
        throw new Error(`Error: ${response.status}`);
      }

      const result = await response.json();

      // Close modal and notify parent
      dispatch('imported', {
        count: sitesList.length,
        successful: result.imported || 0,
        failed: result.failed || 0
      });

      resetForm();
    } catch (error) {
      console.error('Error importing sites:', error);
      alert(`Failed to import sites: ${error.message}`);
    } finally {
      isLoading = false;
    }
  }

  function handleClose() {
    resetForm();
    dispatch('close');
  }

  function resetForm() {
    sites = '';
    showModal = false;
  }
</script>

{#if showModal}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
    <div class="bg-[#202020] rounded-lg p-6 w-full max-w-2xl">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl text-gray-200 font-semibold">Mass Import Servers</h2>
      </div>

      <form on:submit|preventDefault={handleSubmit} class="space-y-4">
        <!-- Sites textarea -->
        <div>
          <label class="block text-gray-300 mb-1" for="sites">
            Enter URLs (one per line or separated by commas/semicolons)
          </label>
          <textarea
            id="sites"
            bind:value={sites}
            rows="10"
            required
            autofocus
            class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-green-500 font-mono"
            placeholder="{placeholderText}"
          ></textarea>
          <p class="text-xs text-gray-400 mt-1">
            Enter URLs separated by newlines, commas, or semicolons. You can mix separators as needed.
          </p>
        </div>

        <!-- Settings section -->
        <div class="border-gray-700 pt-4 mt-4">
          <h3 class="text-lg text-gray-200 mb-3">Global Settings</h3>
          <p class="text-sm text-gray-400 mb-4">
            These settings will apply to all imported sites.
          </p>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="space-y-4">
              <CustomCheckbox
                checked={followRedirect}
                on:change={e => followRedirect = e.detail}
                label="Follow Redirects"
              />

              <CustomCheckbox
                checked={allowInsecure}
                on:change={e => allowInsecure = e.detail}
                label="Allow Insecure"
              />
            </div>

            <div>
              <IntervalSlider
                value={intervalValue}
                on:change={e => intervalValue = e.detail}
                label="Check Interval"
              />
            </div>
          </div>
        </div>

        <!-- Form buttons -->
        <div class="flex justify-end gap-3 mt-6 pt-4 border-gray-700">
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
              <svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              Importing...
            {:else}
              Import Servers
            {/if}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
