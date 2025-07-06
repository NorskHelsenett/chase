<!-- MassImportModal.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import CustomCheckbox from '../ui/CustomCheckbox.svelte';
  import IntervalSlider from '../ui/IntervalSlider.svelte';

  const dispatch = createEventDispatcher();

  export let showModal = false;
  export let isLoading = false;
  
  // Track failed imports
  let showFailedModal = false;
  let failedImports: { url: string, reason: string }[] = [];

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
      
      // Process failed imports if any
      if (result.failed > 0 && result.errors && result.errors.length > 0) {
        failedImports = result.errors.map((error: string) => {
          // Try to extract URL and reason from error message
          // Look for patterns like "Invalid URL format: example.com" or "Server URL already exists: example.com"
          const match = error.match(/^(.*?):\s*(.+?)$/);
          if (match && match[2]) {
            return {
              url: match[2],
              reason: match[1]
            };
          }
          
          // If the extraction pattern didn't work, try to find a URL in the error message
          const urlMatch = error.match(/https?:\/\/[^\s]+|[a-zA-Z0-9][-a-zA-Z0-9.]+\.[a-zA-Z]{2,}/);
          if (urlMatch) {
            const url = urlMatch[0];
            const reason = error.replace(url, '').replace(/:\s*$/, '');
            return {
              url,
              reason: reason.trim() || "Unknown error"
            };
          }
          
          return {
            url: "Unknown URL",
            reason: error
          };
        });
        
        // First close the import modal, then show failed imports modal after a delay
        showModal = false;
        setTimeout(() => {
          showFailedModal = true;
        }, 300);
      } else {
        // Reset form only if there are no failures
        resetForm();
      }

      // Close modal and notify parent
      dispatch('imported', {
        count: sitesList.length,
        successful: result.imported || 0,
        failed: result.failed || 0
      });
    } catch (error) {
      console.error('Error importing sites:', error);
      alert(`Failed to import sites: ${error.message}`);
    } finally {
      isLoading = false;
    }
  }

  function handleClose() {
    // Only reset the form if there are no failed imports waiting to be shown
    if (failedImports.length === 0) {
      resetForm();
    } else {
      // Just close the modal without resetting failed imports
      showModal = false;
    }
    dispatch('close');
  }

  function resetForm() {
    sites = '';
    showModal = false;
    showFailedModal = false;
    failedImports = [];
  }
  
  function closeFailedModal() {
    showFailedModal = false;
    // Now that the user has seen the failures, we can clear them
    failedImports = [];
  }
  
  // Function to add failed imports back to the input
  function retryFailedImports() {
    if (failedImports.length === 0) return;
    
    // Get URLs from failed imports
    const urlsToRetry = failedImports.map(item => item.url);
    
    // Replace the textarea content with just the failed imports
    sites = urlsToRetry.join('\n');
    
    // Close failed modal and show the main import modal
    showFailedModal = false;
    showModal = true;
  }
  
  // Export a function to show failed imports modal from parent
  export function showFailedImportsModal() {
    if (failedImports.length > 0) {
      showFailedModal = true;
    }
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

<!-- Failed Imports Modal -->
{#if showFailedModal}
  <div class="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
    <div class="bg-[#202020] rounded-lg p-6 w-full max-w-2xl max-h-[80vh] flex flex-col">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl text-red-400 font-semibold flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
            <line x1="12" y1="9" x2="12" y2="13"></line>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
          Failed Imports ({failedImports.length})
        </h2>
        <button 
          on:click={closeFailedModal}
          class="p-1 hover:bg-[#333] rounded-lg transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>
      
      <div class="overflow-y-auto flex-grow">
        <table class="w-full text-sm text-left">
          <thead class="bg-[#2b2b2b] text-gray-300">
            <tr>
              <th class="py-2 px-4 rounded-tl-lg">URL</th>
              <th class="py-2 px-4 rounded-tr-lg">Error</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-700">
            {#each failedImports as item}
              <tr class="hover:bg-[#2b2b2b]/50 transition-colors">
                <td class="py-2 px-4 font-mono text-gray-300">{item.url}</td>
                <td class="py-2 px-4 text-red-400">{item.reason}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      
      <div class="mt-4 border-t border-gray-700 pt-4 flex justify-end gap-3">
        <button
          on:click={retryFailedImports}
          class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-green-400 flex items-center gap-2 transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.3"/>
          </svg>
          Retry All
        </button>
        <button
          on:click={closeFailedModal}
          class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 transition-colors"
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
