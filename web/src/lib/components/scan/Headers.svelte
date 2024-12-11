<script>
	import { getRiskColor } from "$lib/utils";
	import { Lock, AlertTriangle } from "lucide-svelte";
	import { fade } from "svelte/transition";

  export let loading = true;
  export let results = {}
</script>

<section>
  <h2 class="text-xl flex items-center gap-2 mb-4">
    <Lock class="w-5 h-5" />
    Security Headers Analysis
  </h2>

  {#if loading}
    <div class="bg-[#202020] rounded-lg p-6 animate-pulse">
      <div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
      <div class="space-y-2">
        {#each Array(3) as _}
          <div class="h-4 bg-gray-700 rounded w-full"></div>
        {/each}
      </div>
    </div>
  {:else if results}
    <div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
      <div class="flex items-center gap-4 mb-6">
        <div
          class="text-4xl font-bold {getRiskColor(results.headers.score)}"
        >
          {results.headers.score}
        </div>
        <div class="text-gray-400">Security Headers Score</div>
      </div>

      {#if results.headers.issues.length > 0}
        <div class="mb-4">
          <h3 class="text-red-400 flex items-center gap-2 mb-2">
            <AlertTriangle class="w-4 h-4" />
            Issues Found
          </h3>
          <ul class="space-y-1 text-gray-300">
            {#each results.headers.issues as issue}
              <li>• {issue.description}</li>
            {/each}
          </ul>
        </div>
      {/if}

      <div>
        <h3 class="text-green-400 mb-2">Passed Checks</h3>
        <ul class="space-y-1 text-gray-300">
          {#each results.headers.passed as check}
            <li>• {check}</li>
          {/each}
        </ul>
      </div>
    </div>
  {/if}
</section>