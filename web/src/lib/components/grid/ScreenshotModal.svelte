<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fade, fly, scale } from 'svelte/transition';
  import { Scale, Globe, FileText, FileSearch, Shield, Server as ServerIcon, AlertTriangle, X, Zap, ArrowLeft, ArrowRight, ExternalLink, Maximize2, Minimize2 } from 'lucide-svelte';
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
  let imageLoaded = false;

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
  <div class="absolute inset-0 bg-black/80 backdrop-blur-md"></div>

  <!-- Modal -->
  <div
    transition:fly={{ y: 20, duration: 300 }}
    class="relative z-10 bg-[#202020] text-white rounded-xl shadow-2xl w-full max-w-7xl max-h-[90vh] m-4 overflow-hidden border border-green-900/30"
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
      <div class="flex justify-between items-center p-4 border-b border-green-900/30 bg-gradient-to-r from-[#1a1a1a] to-[#202020]">
        <h2 class="text-xl font-medium flex items-center gap-2">
          <Globe size={20} class="text-green-400" />
          {getHostname(currentSite.url)}
        </h2>
        <div class="flex items-center gap-2">
          <button
            on:click={() => openSiteUrl(currentSite.url)}
            class="p-2 hover:bg-green-900/30 rounded-lg transition-colors text-green-400 hover:text-green-300"
            title="Open site in new tab"
          >
            <ExternalLink size={20} />
          </button>
          <button
            on:click={toggleFullscreen}
            class="p-2 hover:bg-green-900/30 rounded-lg transition-colors text-green-400 hover:text-green-300"
            title={showingFullscreenImage ? 'Exit fullscreen' : 'Fullscreen'}
          >
            {#if showingFullscreenImage}
              <Minimize2 size={20} />
            {:else}
              <Maximize2 size={20} />
            {/if}
          </button>
          <button
            on:click={closeModal}
            class="p-2 hover:bg-green-900/30 rounded-lg transition-colors"
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
            class="flex-1 p-4 flex items-center justify-center bg-black/80"
            on:click={toggleFullscreen}
            transition:fade={{ duration: 200 }}
          >
            <!-- Loading indicator for image -->
            <div class="absolute inset-0 flex items-center justify-center" class:hidden={imageLoaded}>
              <div class="relative">
                <div class="w-16 h-16 border-4 border-t-green-500 border-r-green-400/40 border-b-green-400/20 border-l-green-400/60 rounded-full animate-spin"></div>
                <div class="absolute inset-0 w-16 h-16 border-4 border-green-500/10 rounded-full animate-pulse"></div>
              </div>
            </div>

            <img
              src={`/api/screenshot/${currentSite.url.replace(/^(https?:\/\/)/, '')}`}
              alt={`Screenshot of ${currentSite.url}`}
              class="w-full h-full object-contain drop-shadow-2xl"
              on:load={() => imageLoaded = true}
              style={!imageLoaded ? 'visibility: hidden' : ''}
            />
          </div>
        {:else}
          <!-- Regular Content View -->
          <!-- Screenshot -->
          <div class="flex-1 p-6 bg-gradient-to-br from-black/40 to-black/60 overflow-hidden">
            <div
              class="w-full h-full flex items-center justify-center relative rounded-lg overflow-hidden"
              on:click={toggleFullscreen}
              title="Click to enlarge"
            >
              <!-- Loading indicator for image -->
              <div class="absolute inset-0 flex items-center justify-center bg-black/30 rounded-lg" class:hidden={imageLoaded}>
                <div class="relative">
                  <div class="w-12 h-12 border-4 border-t-green-500 border-r-green-400/40 border-b-green-400/20 border-l-green-400/60 rounded-full animate-spin"></div>
                  <div class="absolute inset-0 w-12 h-12 border-4 border-green-500/10 rounded-full animate-pulse"></div>
                </div>
              </div>

              <img
                src={`/api/screenshot/${currentSite.url.replace(/^(https?:\/\/)/, '')}`}
                alt={`Screenshot of ${currentSite.url}`}
                class="w-full h-full object-contain rounded-lg shadow-xl"
                on:load={() => imageLoaded = true}
                style={!imageLoaded ? 'visibility: hidden' : ''}
              />
            </div>
          </div>

          <!-- Server Info Sidebar -->
          <div class="w-96 bg-[#1a1a1a] p-4 overflow-y-auto ">
            {#if loading}
              <div class="flex flex-col items-center justify-center h-full space-y-4 py-8">
                <div class="relative">
                  <div class="w-12 h-12 border-4 border-t-green-500 border-r-green-400/40 border-b-green-400/20 border-l-green-400/60 rounded-full animate-spin"></div>
                  <div class="absolute inset-0 w-12 h-12 border-4 border-green-500/10 rounded-full animate-pulse"></div>
                </div>
                <p class="text-green-400">Loading report data...</p>
              </div>
            {:else if error}
              <div in:fade={{ duration: 200 }} class="p-4 bg-red-900/20 rounded-lg border border-red-800/50 text-red-400 flex items-start gap-3">
                <AlertTriangle size={20} class="flex-shrink-0 mt-1" />
                <div>
                  <p class="font-medium">Failed to load report</p>
                  <p class="text-sm mt-1">{error}</p>
                </div>
              </div>
            {:else if currentReport}
              <div class="space-y-5" in:fade={{ duration: 300, delay: 100 }}>
                <!-- URL -->
                <div class="bg-gradient-to-r from-gray-800/50 to-gray-900/50 p-4 rounded-lg border border-green-900/30 shadow-md">
                  <a
                    href={`/server/${currentSite.ID}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex items-center gap-2 text-green-400 hover:text-green-300 transition-colors font-medium group"
                  >
                    <ServerIcon size={18} class="group-hover:scale-110 transition-transform" />
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
                    <div in:fade={{ duration: 200 }} class="bg-red-900/20 p-4 rounded-lg border border-red-800/50 shadow-md">
                      <h3 class="font-semibold mb-2 flex items-center gap-2">
                        <AlertTriangle size={18} class="text-red-400" />
                        Scan Errors
                      </h3>
                      <div class="space-y-2">
                        {#each currentReport.scanErrors as error, i}
                          <p in:fade={{ duration: 200, delay: 100 + i * 50 }} class="text-sm text-red-400 pl-6 border-l border-red-800/30">{error.error}</p>
                        {/each}
                      </div>
                    </div>
                  {:else}


                    <div class="bg-gradient-to-b from-gray-800/30 to-gray-900/30 border border-green-900/30 rounded-lg  shadow-lg overflow-hidden">
                      <div class="grid grid-cols-2 bg-gray-800/50">
                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <Shield size={18} class="text-green-400" />
                          <span class="text-sm font-medium">Header Score</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          <span in:scale={{duration: 200, delay: 200}} class="font-medium px-2 py-1 rounded-md bg-gray-800/80 shadow-inner {getScoreColor(currentReport.headers?.score)}">{currentReport.headers?.score || 'N/A'}</span>
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <Scale size={18} class="text-green-400" />
                          <span class="text-sm font-medium">Certificate</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          <span in:scale={{duration: 200, delay: 250}} class="font-medium px-2 py-1 rounded-md bg-gray-800/80 shadow-inner {getScoreColor(currentReport.certificate?.grade)}">{currentReport.certificate?.grade || 'N/A'}</span>
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <ServerIcon size={18} class="text-green-400" />
                          <span class="text-sm font-medium">Infrastructure</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          <span class="font-mono text-sm text-gray-300">{currentReport.infrastructure?.ip || 'N/A'}</span>
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <Zap size={18} class="text-green-400" />
                          <span class="text-sm font-medium">Status</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          <span in:scale={{duration: 200, delay: 300}} class="font-medium px-2 py-1 rounded-md bg-gray-800/80 shadow-inner {getStatusColor(currentReport.infrastructure?.status)}">
                            {getStatusCode(currentReport.infrastructure?.status)}
                          </span>
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <FileText size={18} class="text-green-400" />
                          <span class="text-sm font-medium">robots.txt</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          {#if currentReport.robotsTxt?.exists}
                            <span in:scale={{duration: 200, delay: 350}} class="text-green-400 bg-green-500/10 p-1 rounded-full">
                              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
                            </span>
                          {:else}
                            <span in:scale={{duration: 200, delay: 350}} class="text-red-400 bg-red-500/10 p-1 rounded-full">
                              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
                            </span>
                          {/if}
                        </div>

                        <div class="flex items-center gap-2 p-3 bg-[#1a1a1a]">
                          <FileSearch size={18} class="text-green-400" />
                          <span class="text-sm font-medium">security.txt</span>
                        </div>
                        <div class="p-3 bg-[#1a1a1a] flex justify-end items-center">
                          {#if currentReport.securityTxt?.exists}
                            <span in:scale={{duration: 200, delay: 400}} class="text-green-400 bg-green-500/10 p-1 rounded-full">
                              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
                            </span>
                          {:else}
                            <span in:scale={{duration: 200, delay: 400}} class="text-red-400 bg-red-500/10 p-1 rounded-full">
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
      <div class="flex justify-between items-center p-4 border-t border-green-900/30 bg-gradient-to-r from-[#1a1a1a] to-[#202020]">
        <button
          class="flex items-center gap-2 px-4 py-2 bg-green-900/30 hover:bg-green-800/50 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors shadow-sm"
          disabled={currentIndex === 0}
          on:click={() => navigateImage(-1)}
        >
          <ArrowLeft size={16} />
          <span>Previous</span>
        </button>
        <span class="font-medium bg-black/30 px-3 py-1 rounded-full text-sm">{currentIndex + 1} of {sites.length}</span>
        <button
          class="flex items-center gap-2 px-4 py-2 bg-green-900/30 hover:bg-green-800/50 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors shadow-sm"
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
