<!-- MassImportModal.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import CustomCheckbox from '../ui/CustomCheckbox.svelte';
  import IntervalSlider from '../ui/IntervalSlider.svelte';
  import ToggleButton from '../ui/ToggleButton.svelte';

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
  let isActive = true; // Default active status
  let updateExisting = false; // Default to not update existing servers
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
          active: isActive,
        },
        update_existing: updateExisting // Add the update option to the request
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
        failed: result.failed || 0,
        failedImports: failedImports // Pass the failed imports to parent
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

  export function resetForm() {
    sites = '';
    intervalValue = 15; // Reset to default
    followRedirect = true; // Reset to default
    allowInsecure = false; // Reset to default
    isActive = true; // Reset to default
    updateExisting = false; // Reset to default
    showFailedModal = false;
    failedImports = [];
    // Don't close the modal if called externally
    if (showModal) {
      showModal = false;
    }
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

  // Show failed imports with specified data from parent
  export function showFailedImports(imports) {
    if (imports && imports.length > 0) {
      // Set the failed imports
      failedImports = imports;
      // Show the modal
      showFailedModal = true;
      return true;
    }
    return false;
  }
</script>

{#if showModal}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
    <div class="bg-[#202020] rounded-lg p-6 w-full max-w-2xl">
      <div class="flex items-center justify-between border-gray-700 pb-4">
        <h2 class="text-2xl text-gray-200 font-semibold flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path>
            <polyline points="17 21 17 13 7 13 7 21"></polyline>
            <polyline points="7 3 7 8 15 8"></polyline>
          </svg>
          Mass Import Servers
        </h2>
      </div>

      <form on:submit|preventDefault={handleSubmit} class="space-y-6">
        <!-- Sites textarea -->
        <div class="">
          <label class="block text-gray-300 mb-2 font-medium" for="sites">
            Enter URLs (one per line or separated by commas/semicolons)
          </label>
          <textarea
            id="sites"
            bind:value={sites}
            rows="8"
            required
            autofocus
            class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 focus:outline-none focus:ring-1 focus:ring-green-500 font-mono"
            placeholder="{placeholderText}"
          ></textarea>
          <p class="text-xs text-gray-400 mt-2">
            Enter URLs separated by newlines, commas, or semicolons. You can mix separators as needed.
          </p>
        </div>

        <!-- Settings section -->
        <div class="border-gray-700">
          <div class="flex items-center mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="mr-2 text-green-500">
              <circle cx="12" cy="12" r="3"></circle>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
            </svg>
            <h2 class="text-lg text-gray-200 font-medium">Global Settings</h2>
          </div>
          <p class="text-sm text-gray-400 mb-5 ml-6">
            These settings will apply to all imported sites.
          </p>

          <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 bg-[#1e1e1e] p-4 rounded-lg">
            <!-- Left column: Checkboxes -->
            <div class="space-y-4">
              <h4 class="text-sm text-gray-300 font-medium mb-3">Options</h4>
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

            <!-- Middle column: Server Status -->
            <div class="">
              <h4 class="text-sm text-gray-300 font-medium mb-3">Server Status</h4>
              <div class="mt-2 flex-grow flex items-center">
                <ToggleButton
                  bind:value={isActive}
                  on:change={e => isActive = e.detail}
                  width="w-36"
                  disabled={isLoading}
                />
              </div>
            </div>

            <!-- Right column: Interval Slider -->
            <div class="flex flex-col">
              <h4 class="text-sm text-gray-300 font-medium mb-3">Check Interval</h4>
              <div class="flex-grow">
                <IntervalSlider
                  value={intervalValue}
                  on:change={e => intervalValue = e.detail}
                  label="Check Interval"
                />
              </div>
            </div>
          </div>
          <div class="mt-4 ml-4">
              <CustomCheckbox
                checked={updateExisting}
                on:change={e => updateExisting = e.detail}
                label="Update Existing Servers"
              />
              {#if updateExisting}
                <p class="text-xs text-amber-400 mt-1 pl-7">
                  Existing servers will have their settings updated to match these global settings
                </p>
              {/if}
          </div>
        </div>

        <!-- Form buttons -->
        <div class="border-gray-700 pt-5 mt-6 flex justify-end gap-4">
          <button
            type="button"
            on:click={handleClose}
            disabled={isLoading}
            class="px-5 py-2.5 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 transition-colors disabled:opacity-50 disabled:cursor-not-allowed font-medium"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isLoading}
            class="px-5 py-2.5 bg-green-600 hover:bg-green-700 rounded-lg text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2 font-medium min-w-[140px] justify-center"
          >
            {#if isLoading}
              <svg class="animate-spin h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <span>Importing...</span>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="mr-1">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                <polyline points="7 10 12 15 17 10"></polyline>
                <line x1="12" y1="15" x2="12" y2="3"></line>
              </svg>
              {updateExisting ? 'Import/Update Servers' : 'Import Servers'}
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
    <div class="bg-[#202020] rounded-lg p-6 w-full max-w-3xl max-h-[80vh] flex flex-col">
      <div class="flex items-center justify-between mb-6 border-gray-700 pb-4">
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
          class="p-2 hover:bg-[#333] rounded-lg transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <div class="overflow-y-auto flex-grow bg-[#1e1e1e] rounded-lg">
        <table class="w-full text-sm text-left">
          <thead class="bg-[#2b2b2b] text-gray-300 sticky top-0">
            <tr>
              <th class="py-3 px-4 rounded-tl-lg font-medium">URL</th>
              <th class="py-3 px-4 rounded-tr-lg font-medium">Error</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-700">
            {#each failedImports as item}
              <tr class="hover:bg-[#2b2b2b]/50 transition-colors">
                <td class="py-2.5 px-4 font-mono text-gray-300 break-all">{item.url}</td>
                <td class="py-2.5 px-4 text-red-400">{item.reason}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="mt-5 border-gray-700 pt-5 flex justify-end gap-4">
        <button
          on:click={retryFailedImports}
          class="px-5 py-2.5 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-green-400 flex items-center gap-2 transition-colors font-medium"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.3"/>
          </svg>
          Retry All
        </button>
        <button
          on:click={closeFailedModal}
          class="px-5 py-2.5 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 transition-colors font-medium"
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
