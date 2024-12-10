<script>
  import { Clock, Share, Trash2 } from 'lucide-svelte';
  import { fade, fly } from 'svelte/transition';
  import { onMount, onDestroy } from 'svelte';
  import { searchHistory } from '$lib/stores/searchStore';
  import { getRelativeTime } from '$lib/utils/time';
  
  /** @type {import('./$types').PageData} */
  export let data;
  
  let loading = true;
  let searchResults = [];
  let searchTimestamp = Date.now();
  let relativeTime = 'now';
  let timeInterval;

  function updateRelativeTime() {
      relativeTime = getRelativeTime(searchTimestamp);
  }

  onMount(() => {
      timeInterval = setInterval(updateRelativeTime, 1000);
  });

  onDestroy(() => {
      if (timeInterval) clearInterval(timeInterval);
  });
  
  async function fetchSearchResults(query) {
      loading = true;
      searchTimestamp = Date.now();
      try {
          // Simulate API delay
          await new Promise(resolve => setTimeout(resolve, 1500));
          const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
          const data = await response.json();
          searchResults = data;
          // Store search in history
          searchHistory.addSearch(query, data);
      } catch (error) {
          console.error('Error fetching search results:', error);
          searchResults = [];
      } finally {
          loading = false;
      }
  }

  $: if (data.query) {
      fetchSearchResults(data.query);
  }
</script>

<div class="min-h-screen w-full text-gray-100">
  <div class="border-b border-[#2a2a2a]">
      <!-- Title Bar -->
      <div class="pb-2">
          <div class="flex justify-between gap-4 text-gray-400">
              <div class="flex items-center gap-1">
                  <Clock class="w-4 h-4" />
                  <span class="text-sm">{relativeTime}</span>
              </div>
              <span class="text-xl text-gray-500 truncate max-w-[25em]">{data.query}</span>
              <div class="flex items-center gap-2">
                <Share class="w-4 h-4" />
                <Trash2 class="w-4 h-4 mr-2 alert" />
              </div>
          </div>
      </div>
  </div>

  <div class="max-w-3xl mx-auto p-4">
    <h1 class="text-2xl pb-4 pt-4">{data.query}</h1>
      <!-- Source Section -->
      <div class="text-xl flex items-center gap-2 mb-6">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/>
              <path d="M12 15a3 3 0 100-6 3 3 0 000 6z"/>
              <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
          </svg>
          <span>Sources</span>
      </div>

      <!-- Results/Loading States -->
      <div class="space-y-4">
          {#if loading}
              <!-- Loading states remain the same -->
              {#each Array(3) as _, i}
                  <div 
                      class="bg-[#202020] rounded-lg p-4 animate-pulse"
                      in:fade|local={{ duration: 200, delay: i * 100 }}
                  >
                      <div class="flex items-center gap-4">
                          <div class="w-12 h-12 bg-gray-700 rounded-lg"></div>
                          <div class="flex-1">
                              <div class="h-4 bg-gray-700 rounded w-3/4 mb-2"></div>
                              <div class="h-3 bg-gray-700 rounded w-1/2"></div>
                          </div>
                      </div>
                      <div class="mt-4 h-4 bg-gray-700 rounded w-full"></div>
                      <div class="mt-2 h-4 bg-gray-700 rounded w-5/6"></div>
                  </div>
              {/each}
          {:else}
              {#each searchResults as result, i}
                  <div 
                      class="bg-[#202020] hover:bg-[#252525] transition-colors rounded-lg p-4"
                      in:fade|local={{ duration: 200, delay: i * 100 }}
                  >
                      <div class="flex items-center gap-4">
                          <div class="w-12 h-12 bg-[#2b2b2b] rounded-lg flex items-center justify-center">
                              <img 
                                  src={result.icon || "/api/placeholder/48/48"} 
                                  alt=""
                                  class="w-8 h-8 rounded"
                              />
                          </div>
                          <div>
                              <h3 class="text-[#4cc9f0] hover:underline">
                                  <a href={result.url} target="_blank" rel="noopener">
                                      {result.title}
                                  </a>
                              </h3>
                              <span class="text-sm text-gray-400">{result.domain}</span>
                          </div>
                      </div>
                      <p class="mt-3 text-gray-300 line-clamp-2">{result.description}</p>
                  </div>
              {/each}
          {/if}
      </div>
  </div>
</div>

<style>
  @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: .5; }
  }
  .animate-pulse {
      animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
  }
</style>