<!-- MassImport.svelte -->
<script lang="ts">
  import { fade } from 'svelte/transition';
  import MassImportModal from './MassImportModal.svelte';
  
  let showImportModal = false;
  let isLoading = false;
  let importStats = null;
  let showStats = false;
  let importModalComponent: MassImportModal;
  
  function handleImported(event) {
    importStats = event.detail;
    showStats = true;
    setTimeout(() => {
      showStats = false;
    }, 5000);
  }
  
  function showFailedImports() {
    if (importStats && importStats.failed > 0 && importModalComponent) {
      importModalComponent.showFailedImportsModal();
    }
  }
</script>

<div class="space-y-6">
  <div class="bg-[#202020] rounded-lg p-4">
    <div class="flex items-center justify-between">
      <h2 class="text-xl text-gray-200 font-semibold">Mass Import</h2>
      
      <button
        on:click={() => showImportModal = true}
        disabled={isLoading}
        class="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white flex items-center gap-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
        </svg>
        Import Servers
      </button>
    </div>
    
    <p class="mt-2 text-gray-400 text-sm">
      Add multiple servers at once with shared monitoring settings.
    </p>
    
    {#if showStats}
      <div transition:fade={{ duration: 300 }} class="mt-4 p-3 bg-green-900/30 border border-green-800/50 rounded-lg">
        <div class="flex items-center">
          <svg class="w-5 h-5 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <p class="text-green-300">
            Imported {importStats.successful} of {importStats.count} servers successfully.
            {#if importStats.failed > 0}
              <span class="text-yellow-300 hover:underline cursor-pointer" on:click={showFailedImports}>
                {importStats.failed} failed. Click to view details.
              </span>
            {/if}
          </p>
        </div>
      </div>
    {/if}
  </div>
</div>

<MassImportModal 
  bind:showModal={showImportModal} 
  bind:isLoading 
  bind:this={importModalComponent}
  on:imported={handleImported} 
  on:close={() => showImportModal = false}
/>
