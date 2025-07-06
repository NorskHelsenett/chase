<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { Scale, Globe, FileText, FileSearch, Shield, Server as ServerIcon, AlertTriangle, X, Zap, ArrowLeft, ArrowRight, ExternalLink } from 'lucide-svelte';
  import type { Server } from '$lib/models';

  export let sites: Server[] = [];
  export let currentIndex: number;
  export let onClose: () => void;

  let modalOpen = false;
  let currentReport: any = null;
  let loading = false;
  let error: string | null = null;
  let focusTrap: HTMLInputElement;
  let showingFullscreenImage = false;

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
        if (showingFullscreenImage) {
          showingFullscreenImage = false;
        } else {
          closeModal();
        }
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

  function toggleFullscreen() {
    showingFullscreenImage = !showingFullscreenImage;
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

  function openSiteUrl(url: string) {
    // Ensure the URL has a protocol
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
      url = 'https://' + url;
    }

    // Open the URL in a new tab
    window.open(url, '_blank', 'noopener,noreferrer');
  }
</script>

{#if modalOpen}
<div
  transition:fade={{ duration: 200 }}
  class="fixed inset-0 z-50 flex items-center justify-center"
  on:click|self={closeModal}
>
  <!-- Backdrop -->
  <div class="absolute inset-0 bg-black/80 backdrop-blur-sm"></div>

  <!-- Modal -->
  <div
    transition:fly={{ y: 20, duration: 300 }}
    class="relative z-10 bg-[#1a1a1a] text-white rounded-xl shadow-2xl w-full max-w-7xl max-h-[90vh] m-4 overflow-hidden border border-gray-800"
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
      <div class="flex justify-between items-center p-4 border-b border-gray-800">
        <h2 class="text-xl font-medium flex items-center gap-2">
          <Globe size={18} class="text-blue-400" />
          {getHostname(currentSite.url)}
        </h2>
        <div class="flex items-center gap-2">
          <button
            on:click={() => openSiteUrl(currentSite.url)}
            class="p-2 hover:bg-gray-800 rounded-lg transition-colors text-blue-400 hover:text-blue-300"
            title="Open site in new tab"
          >
            <ExternalLink size={20} />
          </button>
          <button
            on:click={closeModal}
            class="p-2 hover:bg-gray-800 rounded-lg transition-colors"
            title="Close"
          >
            <X size={20} />
          </button>
        </div>
      </div>

      <!-- Content -->
      <div class="flex flex-1 min-h-0 overflow-hidden">
        {#if showingFullscreenImage}
          <!-- Fullscreen Screenshot View -->
          <div
            class="flex-1 p-4 flex items-center justify-center bg-black/50 cursor-zoom-out"
            on:click={toggleFullscreen}
          >
            <img
              src={`/api/screenshot/${currentSite.url.replace(/^(https?:\/\/)/, '')}`}
              alt={`Screenshot of ${currentSite.url}`}
              class="w-full h-full object-contain"
            />
          </div>
        {:else}
          <!-- Regular Content View -->
          <!-- Screenshot -->
          <div class="flex-1 p-4 bg-black/30 overflow-hidden">
            <div
              class="w-full h-full flex items-center justify-center cursor-zoom-in"
              on:click={toggleFullscreen}
              title="Click to enlarge"
            >
              <img
                src={`/api/screenshot/${currentSite.url.replace(/^(https?:\/\/)/, '')}`}
                alt={`Screenshot of ${currentSite.url}`}
                class="w-full h-full object-contain rounded-lg shadow-lg"
              />
            </div>
          </div>

          <!-- Server Info Sidebar -->
          <div class="w-96 bg-[#1a1a1a] p-4 overflow-y-auto border-l border-gray-800">
            {#if loading}
              <div class="flex flex-col items-center justify-center h-full space-y-4 py-8">
                <div class="w-8 h-8 border-4 border-t-blue-500 rounded-full animate-spin"></div>
                <p class="text-gray-400">Loading report data...</p>
              </div>
            {:else if error}
              <div class="p-4 bg-red-900/20 rounded-lg border border-red-800/50 text-red-400 flex items-start gap-3">
                <AlertTriangle size={20} class="flex-shrink-0 mt-1" />
                <div>
                  <p class="font-medium">Failed to load report</p>
                  <p class="text-sm mt-1">{error}</p>
                </div>
              </div>
            {:else if currentReport}
              <div class="space-y-5">
                <!-- URL -->
                <div class="bg-gradient-to-r from-gray-800/50 to-gray-900/50 p-4 rounded-lg border border-gray-800">
                  <a
                    href={`/server/${currentSite.ID}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex items-center gap-2 text-blue-400 hover:text-blue-300 transition-colors font-medium"
                  >
                    <ServerIcon size={18} />
                    View Full Server Details
                  </a>
                  <a
                    href="#"
                    on:click|preventDefault={() => openSiteUrl(currentSite.url)}
                    class="mt-2 block text-sm text-gray-300 break-all hover:text-white transition-colors"
                  >
                    <span class="text-gray-500">URL:</span> {currentSite.url}
                  </a>
                </div>

                <!-- Info Grid -->
                <div class="space-y-3">
                  {#if currentReport.scanErrors?.length}
                    <div class="bg-red-900/20 p-4 rounded-lg border border-red-800/50">
                      <h3 class="font-semibold mb-2 flex items-center">
                        <AlertTriangle size={18} class="mr-2 text-red-400" />
                        Scan Errors
                      </h3>
                      {#each currentReport.scanErrors as error}
                        <p class="text-sm text-red-400 mb-1 ml-6">{error.error}</p>
                      {/each}
                    </div>
                  {:else}
                    <h3 class="text-lg font-medium text-gray-300">Security Report</h3>

                    <div class="bg-gray-800/30 rounded-lg border border-gray-800">
                      <div class="grid grid-cols-2 gap-px bg-gray-700">
                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <Shield size={18} class="text-blue-400" />
                          <span class="text-sm">Header Score</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          <span class="font-medium px-2 py-1 rounded-md bg-gray-800 {getScoreColor(currentReport.headers?.score)}">{currentReport.headers?.score || 'N/A'}</span>
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <Scale size={18} class="text-blue-400" />
                          <span class="text-sm">Certificate</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          <span class="font-medium px-2 py-1 rounded-md bg-gray-800 {getScoreColor(currentReport.certificate?.grade)}">{currentReport.certificate?.grade || 'N/A'}</span>
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <ServerIcon size={18} class="text-blue-400" />
                          <span class="text-sm">Infrastructure</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          <span class="font-mono text-sm">{currentReport.infrastructure?.ip || 'N/A'}</span>
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <Zap size={18} class="text-blue-400" />
                          <span class="text-sm">Status</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          <span class="font-medium px-2 py-1 rounded-md bg-gray-800 {getStatusColor(currentReport.infrastructure?.status)}">
                            {getStatusCode(currentReport.infrastructure?.status)}
                          </span>
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <FileText size={18} class="text-blue-400" />
                          <span class="text-sm">robots.txt</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          {#if currentReport.robotsTxt?.exists}
                            <span class="text-green-400">
                              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
                            </span>
                          {:else}
                            <span class="text-red-400">
                              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
                            </span>
                          {/if}
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <FileSearch size={18} class="text-blue-400" />
                          <span class="text-sm">security.txt</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          {#if currentReport.securityTxt?.exists}
                            <span class="text-green-400">
                              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
                            </span>
                          {:else}
                            <span class="text-red-400">
                              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
                            </span>
                          {/if}
                        </div>
                      </div>
                    </div>
                  {/if}
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Navigation Footer -->
      <div class="flex justify-between items-center p-4 border-t border-gray-800">
        <button
          class="flex items-center gap-2 px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          disabled={currentIndex === 0}
          on:click={() => navigateImage(-1)}
        >
          <ArrowLeft size={16} />
          <span>Previous</span>
        </button>
        <span class="font-medium">{currentIndex + 1} of {sites.length}</span>
        <button
          class="flex items-center gap-2 px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          disabled={currentIndex === sites.length - 1}
          on:click={() => navigateImage(1)}
        >
          <span>Next</span>
          <ArrowRight size={16} />
        </button>
      </div>
    </div>
  </div>
</div>
{/if}
