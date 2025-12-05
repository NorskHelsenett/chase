<script lang="ts">
  export let probes: {
    paths?: Record<string, number>;
    findings?: { description: string }[];
  } | null = null;

  const statusClass = (code: number) => {
    if (code >= 500) return 'text-red-400';
    if (code >= 400) return 'text-yellow-400';
    return 'text-green-400';
  };
</script>

{#if probes}
  <div class="bg-[#202020] rounded-lg p-6 space-y-4">
    <div>
      <h3 class="text-gray-400 mb-2">Health Endpoint Status</h3>
      {#if Object.keys(probes.paths || {}).length === 0}
        <div class="text-gray-400">No endpoints checked</div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          {#each Object.entries(probes.paths || {}) as [path, code]}
            <div class="p-3 bg-[#2b2b2b] rounded-lg">
              <div class="text-gray-400 text-sm">{path}</div>
              <div class={`text-lg font-semibold ${statusClass(code)}`}>
                {code}
                {#if code === 200}
                  <span class="text-xs text-gray-400 ml-2">OK</span>
                {:else if code === 404}
                  <span class="text-xs text-gray-400 ml-2">Not Found</span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    {#if probes.findings?.length}
      <div>
        <h3 class="text-yellow-400 mb-2">Findings</h3>
        <ul class="space-y-2">
          {#each probes.findings as finding}
            <li class="text-gray-300">• {finding.description}</li>
          {/each}
        </ul>
      </div>
    {:else}
      <div class="text-green-400">No findings detected.</div>
    {/if}
  </div>
{/if}
