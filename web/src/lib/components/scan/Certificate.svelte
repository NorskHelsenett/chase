<script>
	import { getRiskColor } from "$lib/utils";
	import { Wrench, AlertTriangle } from "lucide-svelte";
	import { fade } from "svelte/transition";

  export let loading = true;
  export let results = {}
</script>
<section>
  <h2 class="text-xl flex items-center gap-2 mb-4">
    <Wrench class="w-5 h-5" />
    SSL/TLS Certificate Analysis
  </h2>

  {#if loading}
    <div class="bg-[#202020] rounded-lg p-6 animate-pulse">
      <div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
      <div class="space-y-2">
        {#each Array(4) as _}
          <div class="h-4 bg-gray-700 rounded w-full"></div>
        {/each}
      </div>
    </div>
  {:else if results}
    <div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
      <div class="flex items-center gap-4 mb-6">
        <div
          class="text-4xl font-bold {getRiskColor(results.certificate.grade)}"
        >
          {results.certificate.grade}
        </div>
        <div class="text-gray-400">Certificate Grade</div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div class="grid grid-cols-[auto,1fr] gap-x-2">
          <h3 class="text-gray-400 mb-2 col-span-2">Certificate Details</h3>

          <p class="text-gray-300">Valid until:</p>
          <p class="text-gray-300">{results.certificate.validUntil}</p>

          <p class="text-gray-300">Organization:</p>
          <p class="text-gray-300">{results.certificate.organization}</p>

          <p class="text-gray-300">Issuer:</p>
          <p class="text-gray-300">{results.certificate.issuer}</p>
        </div>

        <div>
          <h3 class="text-gray-400 mb-2">Findings</h3>
          <ul class="space-y-1 text-gray-300">
            {#each results.certificate.findings as finding}
              <li>• {finding}</li>
            {/each}
          </ul>
        </div>

      {#if results.certificate.warnings.length > 0}
        <div class="mt-4">
          <h3 class="text-yellow-400 flex items-center gap-2 mb-2">
            <AlertTriangle class="w-4 h-4" />
            Warnings
          </h3>
          <ul class="space-y-1 text-gray-300">
            {#each results.certificate.warnings as warning}
              <li>• {warning.evidence}</li>
            {/each}
          </ul>
        </div>
      {/if}

      {#if results.certificate.tlsVersions.length > 0}
        <div class="mt-4">
          <h3 class="text-gray-400 flex items-center gap-2 mb-2">
            TLS supported versions
          </h3>
          <ul class="space-y-1 text-gray-300">
            {#each results.certificate.tlsVersions as versions}
              <li>• {versions}</li>
            {/each}
          </ul>
        </div>
      {/if}
    </div>

    </div>
  {/if}
</section>