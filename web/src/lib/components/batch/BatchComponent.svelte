<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  interface StartBatchResponse {
    job_id: string;
    status: string;
    total: number;
  }

  interface BatchJob {
    id: string;
    status: string;
    total: number;
    completed: number;
    failed: number;
    start_time: string;
    end_time?: string;
    error?: string;
  }

  interface BatchJobsResponse {
    active_jobs: BatchJob[];
    completed_jobs: BatchJob[];
    pagination: {
      limit: number;
      offset: number;
    };
    total: number;
  }

  let activeJobs: BatchJob[] = [];
  let completedJobs: BatchJob[] = [];
  let pollingIntervals: { [key: string]: number } = {};
  let isLoading = false;
  let showErrorModal = false;
  let selectedError: string | null = null;

  function showError(error: string) {
    selectedError = error;
    showErrorModal = true;
  }

  function closeErrorModal() {
    showErrorModal = false;
    selectedError = null;
  }

  function sortJobsByDate(jobs: BatchJob[]): BatchJob[] {
    return [...jobs].sort((a, b) => 
      new Date(b.start_time).getTime() - new Date(a.start_time).getTime()
    );
  }

  // Calculate estimated time remaining based on progress rate
  function calculateETA(job: BatchJob): string {
  if (job.completed === 0 || job.status === 'completed') return '';
  
  const startTime = new Date(job.start_time).getTime();
  const now = Date.now();
  const elapsed = (now - startTime) / 1000; // seconds
  
  // Calculate rate: items completed per second
  const rate = job.completed / elapsed;
  
  // If the rate is too low or invalid, return empty string
  if (rate <= 0) return '';
  
  // Calculate remaining items and time
  const remainingItems = job.total - job.completed;
  const estimatedSecondsLeft = remainingItems / rate;
  
  // Add a sanity check for very large numbers
  if (estimatedSecondsLeft > 24 * 60 * 60) return '';
  
  // Format the time remaining
  if (estimatedSecondsLeft < 60) {
    return `${Math.max(1, Math.round(estimatedSecondsLeft))}s left`;
  } else {
    const minutes = Math.round(estimatedSecondsLeft / 60);
    return `${minutes}m left`;
  }
}

  // Start polling for a specific job
  function startPolling(jobId: string) {
    pollingIntervals[jobId] = window.setInterval(async () => {
      try {
        const response = await fetch(`/api/batch/${jobId}/status`);
        if (!response.ok) throw new Error('Failed to fetch status');
        const jobStatus: BatchJob = await response.json();
        
        activeJobs = activeJobs.map(job => 
          job.id === jobId ? { ...job, ...jobStatus } : job
        );
        
        if (jobStatus.status === 'completed' || jobStatus.status === 'failed') {
          clearInterval(pollingIntervals[jobId]);
          delete pollingIntervals[jobId];
          await refreshJobs();
        }
      } catch (error) {
        console.error('Failed to fetch job status:', error);
      }
    }, 1000);
  }

  async function startBatch() {
    isLoading = true;
    try {
      const response = await fetch('/api/batch/start', { 
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      });
      if (!response.ok) throw new Error('Failed to start batch');
      const data: StartBatchResponse = await response.json();
      
      const newJob: BatchJob = {
        id: data.job_id,
        status: 'running',
        total: data.total,
        completed: 0,
        failed: 0,
        start_time: new Date().toISOString(),
      };
      
      activeJobs = [newJob, ...activeJobs];
      startPolling(newJob.id);
    } catch (error) {
      console.error('Failed to start batch:', error);
    } finally {
      isLoading = false;
    }
  }

  async function cancelBatch(jobId: string) {
    try {
      const response = await fetch(`/api/batch/${jobId}/cancel`, { method: 'POST' });
      if (!response.ok) throw new Error('Failed to cancel batch');
      clearInterval(pollingIntervals[jobId]);
      delete pollingIntervals[jobId];
      await refreshJobs();
    } catch (error) {
      console.error('Failed to cancel batch:', error);
    }
  }

  async function refreshJobs() {
    try {
      const response = await fetch('/api/batch/list');
      if (!response.ok) throw new Error('Failed to fetch batch list');
      const data: BatchJobsResponse = await response.json();
      
      activeJobs = sortJobsByDate(data.active_jobs);
      completedJobs = sortJobsByDate(data.completed_jobs);
      
      data.active_jobs.forEach(job => {
        if (job.status === 'running' && !pollingIntervals[job.id]) {
          startPolling(job.id);
        }
      });
    } catch (error) {
      console.error('Failed to fetch batch jobs:', error);
    }
  }

  onMount(refreshJobs);

  onDestroy(() => {
    Object.values(pollingIntervals).forEach(interval => clearInterval(interval));
  });
</script>

<div class="relative min-h-[200px]">
  <div class="space-y-6">
    <div class="bg-[#202020] rounded-lg p-4">
      <div class="flex items-center justify-between">
        <h2 class="text-xl text-gray-200 font-semibold">Batch Operations</h2>
        
        <button
          on:click={startBatch}
          disabled={isLoading}
          class="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white flex items-center gap-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          {isLoading ? 'Starting...' : 'Start New Batch'}
        </button>
      </div>
    </div>

    <div class="space-y-4">
      {#each activeJobs as job (job.id)}
        <div class="bg-[#202020] rounded-lg p-4 space-y-3">
          <div class="flex justify-between items-center">
            <div class="flex items-center gap-2">
              <div class="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
              <span class="font-medium text-gray-200">Batch {job.id}</span>
            </div>
            
            <button 
              on:click={() => cancelBatch(job.id)}
              class="px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-white flex items-center gap-2 transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
              Cancel
            </button>
          </div>

          <div class="space-y-2">
            <div class="h-2 bg-[#2b2b2b] rounded-full overflow-hidden">
              <div
                class="h-full bg-blue-500 transition-all duration-500"
                style="width: {(job.completed / job.total) * 100}%"
              />
            </div>
            
            <div class="flex justify-between text-sm text-gray-400">
              <div class="flex items-center gap-4">
                <span>{job.completed} / {job.total} completed</span>
                {#if job.failed > 0}
                  <!-- svelte-ignore a11y-click-events-have-key-events -->
                  <!-- svelte-ignore a11y-no-static-element-interactions -->
                  <span 
                    class="text-red-400 flex items-center gap-1 cursor-pointer hover:text-red-300 transition-colors"
                    on:click|stopPropagation={() => showError(job.error || 'No error details available')}
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    {job.failed} failed
                  </span>
                {/if}
              </div>
              
              <div class="flex items-center gap-1">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{calculateETA(job)}</span>
              </div>
            </div>
          </div>
        </div>
      {/each}

      {#each completedJobs as job (job.id)}
        <div class="bg-[#202020] rounded-lg p-4 space-y-3">
          <div class="flex justify-between items-center">
            <div class="flex items-center gap-2">
              {#if job.status === 'completed'}
                <svg class="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              {:else if job.status === 'failed'}
                <svg class="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              {/if}
              <span class="font-medium text-gray-200">Batch {job.id}</span>
            </div>
            
            <span class="text-sm text-gray-400">
              {new Date(job.end_time || '').toLocaleString()}
            </span>
          </div>

          <div class="space-y-2">
            <div class="h-2 bg-[#2b2b2b] rounded-full overflow-hidden">
              <div
                class="h-full {job.status === 'completed' ? 'bg-green-500' : 'bg-red-500'} transition-all duration-500"
                style="width: 100%"
              />
            </div>
            
            <div class="flex justify-between text-sm text-gray-400">
              <div class="flex items-center gap-4">
                <span>{job.completed} / {job.total} completed</span>
                {#if job.failed > 0}
                  <!-- svelte-ignore a11y-click-events-have-key-events -->
                  <!-- svelte-ignore a11y-no-static-element-interactions -->
                  <span 
                    class="text-red-400 flex items-center gap-1 cursor-pointer hover:text-red-300 transition-colors"
                    on:click|stopPropagation={() => showError(job.error || 'No error details available')}
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    {job.failed} failed
                  </span>
                {/if}
              </div>
            </div>
          </div>
        </div>
      {/each}

      {#if activeJobs.length === 0 && completedJobs.length === 0}
        <div class="text-center text-gray-400 py-8">
          No batch operations found. Start a new batch to begin processing.
        </div>
      {/if}
    </div>
  </div>

  <!-- Info button in bottom right -->
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div 
    class="fixed bottom-4 right-4 p-2 bg-[#202020] rounded-full shadow-lg cursor-pointer hover:bg-[#2b2b2b] transition-colors"
    on:click={() => showError('Error logs for batch operations:\n\n' + 
      completedJobs
        .filter(job => job.error)
        .map(job => `Batch ${job.id}:\n${job.error}`)
        .join('\n\n')
    )}
  >
    <svg class="w-6 h-6 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  </div>
</div>

<!-- Error Modal -->
{#if showErrorModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center"
    on:click|self={closeErrorModal}
  >
    <!-- Backdrop -->
    <div class="absolute inset-0 bg-black/70"></div>

    <!-- Modal -->
    <div
      class="relative z-10 bg-[#202020] text-white rounded-lg shadow-xl w-full max-w-lg m-4"
      role="dialog"
      aria-modal="true"
    >
    <!-- Header -->
    <div class="flex justify-between items-center p-4 border-b border-[#2b2b2b]">
      <h2 class="text-xl font-semibold">Error Details</h2>
      <button
        on:click={closeErrorModal}
        class="p-2 hover:bg-[#333] rounded-lg transition-colors"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Content -->
    <div class="p-4">
      <div class="bg-[#2b2b2b] p-4 rounded-lg">
        <pre class="text-sm text-red-400 whitespace-pre-wrap">{selectedError}</pre>
      </div>
    </div>

    <!-- Footer -->
    <div class="flex justify-end p-4 border-t border-[#2b2b2b]">
      <button
        on:click={closeErrorModal}
        class="px-4 py-2 bg-[#2b2b2b] hover:bg-[#333] rounded-lg transition-colors"
      >
        Close
      </button>
    </div>
  </div>
</div>
{/if}