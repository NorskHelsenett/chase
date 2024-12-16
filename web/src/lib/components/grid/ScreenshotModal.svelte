<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Scale, Globe, FileText, FileSearch, Shield, Server as ServerIcon, AlertTriangle, X, Zap } from 'lucide-svelte';
  import type { Server } from '$lib/models';

  export let sites: Server[] = [];
  export let currentIndex: number;
  export let onClose: () => void;

  let modalOpen = false;
  let currentReport: any = null;
  let loading = false;
  let error: string | null = null;
  let focusTrap: HTMLInputElement;

  // Store original overflow style
  let originalOverflow: string;

  async function fetchServerReport(serverId: string) {
    loading = true;
    error = null;
    try {
      const res = await fetch(`/api/servers/${serverId}/report`);
      if (!res.ok) throw new Error('Failed to fetch report');
      currentReport = await res.json();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (!modalOpen) return;

    switch(event.key) {
      case 'ArrowLeft':
        navigateImage(-1);
        break;
      case 'ArrowRight':
        navigateImage(1);
        break;
      case 'Escape':
        closeModal();
        break;
    }
  }

  function navigateImage(direction: number) {
    const newIndex = currentIndex + direction;
    if (newIndex >= 0 && newIndex < sites.length) {
      currentIndex = newIndex;
      fetchServerReport(sites[currentIndex].ID);
    }
  }

  function getHostname(url: string): string {
    try {
      return new URL(url).hostname;
    } catch {
      return url;
    }
  }

  function closeModal() {
    modalOpen = false;
    document.body.style.overflow = originalOverflow;
    onClose();
  }

  onMount(() => {
    modalOpen = true;
    originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    fetchServerReport(sites[currentIndex].ID);
    window.addEventListener('keydown', handleKeydown);
    focusTrap?.focus();
  });

  onDestroy(() => {
    document.body.style.overflow = originalOverflow;
    window.removeEventListener('keydown', handleKeydown);
  });

  $: currentSite = sites[currentIndex];

  function getStatusIcon(value: any, type: string) {
    switch(type) {
      case 'exists':
        return value ? '✓' : '✗';
      case 'exposed':
        return value?.length > 0 ? '⚠️' : '✓';
      default:
        return value;
    }
  }

  function getScoreColor(score: string) {
    switch (score) {
      case "A+":
        return "text-emerald-500";
      case "A":
        return "text-green-500";
      case "B":
        return "text-blue-500";
      case "C":
        return "text-yellow-500";
      case "D":
        return "text-orange-500";
      case "E":
      case "F":
        return "text-red-500";
      default:
        return "text-gray-500";
    }
  }

  function getStatusCode(status: string): string {
  // Extract just the numeric part from strings like "200 OK" or "404 Not Found"
  const match = status?.match(/^\d+/);
  return match ? match[0] : 'N/A';
}

function getStatusColor(status: string): string {
  const code = parseInt(getStatusCode(status));
  if (isNaN(code)) return "text-gray-500";

  if (code >= 200 && code < 300) return "text-green-500";
  if (code >= 300 && code < 400) return "text-blue-500";
  if (code >= 400 && code < 500) return "text-orange-500";
  if (code >= 500) return "text-red-500";
  return "text-gray-500";
}
</script>

{#if modalOpen}
<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
  class="fixed inset-0 z-50 flex items-center justify-center"
  on:click|self={closeModal}
>
  <!-- Backdrop -->
  <div class="absolute inset-0 bg-black/70"></div>

  <!-- Modal -->
  <div
    class="relative z-10 bg-[#202020] text-white rounded-lg shadow-xl w-full max-w-7xl max-h-[90vh] m-4 overflow-hidden"
    role="dialog"
    aria-modal="true"
  >
    <!-- Hidden focus trap -->
    <input
      bind:this={focusTrap}
      type="text"
      class="sr-only"
      tabindex="0"
      aria-hidden="true"
    />

    <div class="flex flex-col h-full">
      <!-- Header -->
      <div class="flex justify-between items-center p-4">
        <h2 class="text-xl font-semibold">{getHostname(currentSite.url)}</h2>
        <button
          on:click={closeModal}
          class="p-2 hover:bg-[#333] rounded-lg transition-colors"
        >
          <X size={24} />
        </button>
      </div>

      <!-- Content -->
      <div class="flex flex-1 min-h-0">
        <!-- Screenshot -->
        <div class="flex-1 p-4">
          <img
            src={`/api/screenshot/${currentSite.url.replace(/^(https?:\/\/)/, '')}`}
            alt={`Screenshot of ${currentSite.url}`}
            class="w-full h-full object-contain rounded-lg"
          />
        </div>

        <!-- Server Info -->
        <div class="w-80 bg-[#202020] p-4 overflow-y-auto">
          {#if loading}
            <div class="animate-pulse space-y-4">
              <div class="h-4 bg-gray-700 rounded w-3/4"></div>
              <div class="h-4 bg-gray-700 rounded w-1/2"></div>
            </div>
          {:else if error}
            <div class="text-red-400">
              <AlertTriangle size={20} class="inline-block mr-2" />
              {error}
            </div>
          {:else if currentReport}
            <div class="space-y-4">
              <!-- URL -->
              <a
                href={`/server/${currentSite.ID}`}
                target="_blank"
                rel="noopener noreferrer"
                class="block text-blue-400 hover:underline break-all"
              >
                <Globe size={20} class="inline-block mr-2" />
                {currentSite.url}
              </a>

              <!-- Info Grid -->
              <div class="grid gap-3">
                {#if currentReport.scanErrors?.length}
                  <div class="bg-red-900/20 p-3 rounded-lg">
                    <h3 class="font-semibold mb-2 flex items-center">
                      <AlertTriangle size={20} class="mr-2" />
                      Scan Errors
                    </h3>
                    {#each currentReport.scanErrors as error}
                      <p class="text-sm text-red-400 mb-1">{error.error}</p>
                    {/each}
                  </div>
                {:else}
                  <div class="grid grid-cols-2 gap-2 items-center p-3 bg-[#2b2b2b] rounded-lg">
                    <div class="flex items-center gap-2">
                      <Shield size={20} />
                      Header Score
                    </div>
                    <span class="text-right {getScoreColor(currentReport.headers?.score)}">{currentReport.headers?.score || 'N/A'}</span>
                  </div>

                  <div class="grid grid-cols-2 gap-2 items-center p-3 bg-[#2b2b2b] rounded-lg">
                    <div class="flex items-center gap-2">
                      <Scale size={20} />
                      Certificate
                    </div>
                    <span class="text-right {getScoreColor(currentReport.certificate?.grade)}">{currentReport.certificate?.grade || 'N/A'}</span>
                  </div>

                  <div class="grid grid-cols-2 gap-2 items-center p-3 bg-[#2b2b2b] rounded-lg">
                    <div class="flex items-center gap-2">
                      <ServerIcon size={20} />
                      Infrastructure
                    </div>
                    <span class="text-right">{currentReport.infrastructure?.ip || 'N/A'}</span>
                  </div>

                  <div class="grid grid-cols-2 gap-2 items-center p-3 bg-[#2b2b2b] rounded-lg">
                    <div class="flex items-center gap-2">
                      <Zap size={20} />
                      Status
                    </div>
                    <span class="text-right {getStatusColor(currentReport.infrastructure?.status)}">
                      {getStatusCode(currentReport.infrastructure?.status)}
                    </span>
                  </div>

                  <div class="grid grid-cols-2 gap-2 items-center p-3 bg-[#2b2b2b] rounded-lg">
                    <div class="flex items-center gap-2">
                      <FileText size={20} />
                      robots.txt
                    </div>
                    <span class="text-right">{getStatusIcon(currentReport.robotsTxt?.exists, 'exists')}</span>
                  </div>

                  <div class="grid grid-cols-2 gap-2 items-center p-3 bg-[#2b2b2b] rounded-lg">
                    <div class="flex items-center gap-2">
                      <FileSearch size={20} />
                      security.txt
                    </div>
                    <span class="text-right">{getStatusIcon(currentReport.securityTxt?.exists, 'exists')}</span>
                  </div>
                {/if}
              </div>
            </div>
          {/if}
        </div>
      </div>

      <!-- Navigation Footer -->
      <div class="flex justify-between items-center p-4">
        <button
          class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={currentIndex === 0}
          on:click={() => navigateImage(-1)}
        >
          Previous
        </button>
        <span>{currentIndex + 1} / {sites.length}</span>
        <button
          class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={currentIndex === sites.length - 1}
          on:click={() => navigateImage(1)}
        >
          Next
        </button>
      </div>
    </div>
  </div>
</div>
{/if}