<script lang="ts">
  import type { Server } from '$lib/models';
  import ScreenshotModal from './ScreenshotModal.svelte';

  export let sites: Server[] = [];

  let imageStates: Record<string, boolean> = {};
  let imageErrors: Record<string, boolean> = {};
  let selectedImageIndex: number | null = null;

  function handleImageLoad(url: string) {
    imageStates[url] = true;
  }

  function handleImageError(url: string) {
    imageErrors[url] = true;
    imageStates[url] = true;
  }

  function getScreenshotUrl(url: string) {
    const cleanUrl = url.replace(/^(https?:\/\/)/, '').replace(/\/$/, '');
    return `/api/screenshot/${cleanUrl}?cached=true`;
  }

  function openModal(index: number) {
    selectedImageIndex = index;
  }

  function closeModal() {
    selectedImageIndex = null;
  }

  function handleClick(event: MouseEvent, site: Server, index: number) {
    // Check if cmd (Mac) or ctrl (Windows/Linux) key is pressed
    if (event.metaKey || event.ctrlKey) {
      openSiteUrl(site.url);
    } else {
      // Normal click behavior - open the modal
      openModal(index);
    }
  }
  
  function handleKeyDown(event: KeyboardEvent, site: Server, index: number) {
    // Enter key opens modal, Enter+Ctrl/Cmd opens URL
    if (event.key === 'Enter') {
      if (event.metaKey || event.ctrlKey) {
        openSiteUrl(site.url);
      } else {
        openModal(index);
      }
    }
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

<div class="bg-[#202020] rounded-lg p-4">
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
    {#each sites as site, index}
      {#if site.active}
        <div
          class="relative group rounded-lg transition-all duration-200 hover:ring-2 hover:ring-green-500 overflow-hidden cursor-pointer"
          on:click={(e) => handleClick(e, site, index)}
          on:keydown={(e) => handleKeyDown(e, site, index)}
          title="Click to view, Cmd/Ctrl+Click to open website"
          tabindex="0"
          role="button"
          aria-label="Screenshot of {site.url}. Click to view larger, Cmd or Ctrl click to visit website."
        >
          <div class="relative w-full pb-[56.25%] overflow-hidden">
            {#if !imageStates[site.url]}
              <div class="absolute inset-0 bg-[#2b2b2b] animate-pulse rounded-lg" />
            {/if}

            {#if imageErrors[site.url]}
              <div class="absolute inset-0 flex items-center justify-center bg-[#2b2b2b] text-gray-400 rounded-lg">
                <span>Failed to load screenshot</span>
                <span class="sr-only">for {site.url}</span>
              </div>
            {/if}

            <img
              src={getScreenshotUrl(site.url)}
              alt={`Screenshot of ${site.url}`}
              class="absolute inset-0 w-full h-full object-cover transition-transform duration-200 group-hover:scale-105 rounded-lg [&:not([src])]:hidden"
              on:load={() => handleImageLoad(site.url)}
              on:error={() => handleImageError(site.url)}
              loading="lazy"
            />

            <div class="absolute bottom-0 left-0 right-0 bg-black/75 p-2 transform translate-y-full transition-transform duration-200 group-hover:translate-y-0 rounded-b-lg">
              <p class="text-white text-sm truncate">
                {site.url}
                <span class="text-xs text-gray-400 block">Click to view, Cmd/Ctrl+Click to open site</span>
              </p>
            </div>
          </div>
        </div>
      {/if}
    {/each}
  </div>

  {#if selectedImageIndex !== null}
    <ScreenshotModal
      {sites}
      currentIndex={selectedImageIndex}
      onClose={closeModal}
    />
  {/if}
</div>

<style>
  img{
    color: transparent;
  }
</style>