<script lang="ts">
  import { fade } from 'svelte/transition';
  import { createEventDispatcher } from 'svelte';
  import type { Server } from '$lib/models';
	import CustomCheckbox from '../CustomCheckbox.svelte';

  const dispatch = createEventDispatcher();

  // Modal state
  let showModal = false;
  let searchQuery = '';
  
  // Form state
  let newServer: Server = {
    url: '',
    follow_redirect: true,
    allow_insecure: false,
    expected_status: 200,
    comment: ''
  };
  
  let expectedDown = false; // Toggle for unreachable status

  // Reset form
  function resetForm() {
    newServer = {
      url: '',
      follow_redirect: true,
      allow_insecure: false,
      expected_status: 200,
      comment: ''
    };
    expectedDown = false;
  }

  // Handle form submission
  async function handleSubmit() {
    if (expectedDown) {
      newServer.expected_status = 0;
    }

    try {
      const response = await fetch('/api/servers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(newServer),
      });

      if (response.ok) {
        dispatch('serverAdded');
        showModal = false;
        resetForm();

        setTimeout(() => {
          handleRefresh();
        }, 1000);
      } else {
        console.error('Failed to add server:', await response.text());
      }
    } catch (error) {
      console.error('Error adding server:', error);
    }
  }

  // Handle search
  function handleSearch() {
    dispatch('search', { query: searchQuery });
  }

  // Handle refresh
  function handleRefresh() {
    dispatch('refresh');
  }
</script>

<div class="bg-[#202020] rounded-lg p-4 mb-4">
  <div class="flex items-center justify-between gap-4">
    <!-- Search -->
    <div class="flex-1">
      <input
        type="text"
        bind:value={searchQuery}
        on:input={handleSearch}
        placeholder="Search domains..."
        class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
    </div>

    <!-- Control buttons -->
    <div class="flex gap-3">
      <button
        on:click={handleRefresh}
        class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 flex items-center gap-2 transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        Refresh
      </button>
      
      <button
        on:click={() => showModal = true}
        class="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white flex items-center gap-2 transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Add Server
      </button>
    </div>
  </div>
</div>

<!-- Modal -->
{#if showModal}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" transition:fade>
    <div class="bg-[#202020] rounded-lg p-6 w-full max-w-xl">
      <h2 class="text-xl text-gray-200 font-semibold mb-4">Add New Server</h2>
      
      <form on:submit|preventDefault={handleSubmit} class="space-y-4">
        <div>
          <label class="block text-gray-300 mb-1" for="url">URL</label>
          <input
            id="url"
            type="text"
            bind:value={newServer.url}
            required
            class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div class="flex gap-6">
          <CustomCheckbox
            bind:checked={newServer.follow_redirect}
            label="Follow Redirects"
          />
          
          <CustomCheckbox
            bind:checked={newServer.allow_insecure}
            label="Allow Insecure"
          />
          
          <CustomCheckbox
            bind:checked={expectedDown}
            label="Expected Down"
          />
        </div>

        {#if !expectedDown}
          <div>
            <label class="block text-gray-300 mb-1" for="status">Expected Status Code</label>
            <input
              id="status"
              type="number"
              bind:value={newServer.expected_status}
              min="100"
              max="599"
              class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        {/if}

        <div>
          <label class="block text-gray-300 mb-1" for="comment">Comment</label>
          <textarea
            id="comment"
            bind:value={newServer.comment}
            rows="3"
            class="w-full px-4 py-2 bg-[#2b2b2b] rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
          ></textarea>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button
            type="button"
            on:click={() => { showModal = false; resetForm(); }}
            class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg text-gray-200 transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white transition-colors"
          >
            Add Server
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}