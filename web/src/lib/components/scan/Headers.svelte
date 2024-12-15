<script>
  import { AlertTriangle, Lock, Info, Shield, ShieldAlert, ArrowRight } from "lucide-svelte";
  import { fade, slide } from "svelte/transition";

  export let loading = false;
  export let results = {};

  let expandedIssue = null;

  function getRiskColor(risk) {
    switch (risk) {
      case "CRITICAL":
        return "text-red-500";
      case "HIGH":
        return "text-orange-500";
      case "MEDIUM":
        return "text-yellow-500";
      case "LOW":
        return "text-green-500";
      default:
        return "text-blue-500";
    }
  }

  function getScoreColor(score) {
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
</script>

{#if loading}
  <div class="w-full rounded-lg bg-card border p-6 animate-pulse">
    <div class="flex items-center gap-2 mb-6">
      <Lock class="w-5 h-5" />
      <div class="h-8 bg-muted rounded w-1/3" />
    </div>
    <div class="space-y-4">
      <div class="h-12 bg-muted rounded" />
      <div class="space-y-2">
        {#each Array(3) as _}
          <div class="h-4 bg-muted rounded w-full" />
        {/each}
      </div>
    </div>
  </div>
{:else if results?.headers}
  <div class="w-full rounded-lg bg-card border p-6" in:fade={{ duration: 200 }}>
    <!-- Header -->
    <div class="mb-6">
      <h2 class="text-xl flex items-center gap-2">
        <Lock class="w-5 h-5" />
        Security Headers Analysis
      </h2>
    </div>

    <!-- Score -->
    <div class="flex items-center gap-4 mb-8">
      <div class="text-4xl font-bold {getScoreColor(results.headers.score)}">
        {results.headers.score}
      </div>
      <div class="text-muted-foreground">Security Headers Score</div>
    </div>

    <!-- Issues -->
    {#if results.headers.issues.length > 0}
      <div class="space-y-4 mb-8">
        <h3 class="text-lg font-semibold flex items-center gap-2 text-red-400">
          <AlertTriangle class="w-5 h-5" />
          Security Issues
        </h3>

        <div class="space-y-3">
          {#each results.headers.issues as issue, index}
            <div class="rounded-lg bg-[#2b2b2b]">
              <!-- Issue Header -->
              <button
                class="w-full text-left px-4 py-3 flex items-start gap-2"
                on:click={() => expandedIssue = expandedIssue === index ? null : index}
              >
                <ShieldAlert class="w-5 h-5 mt-1 {getRiskColor(issue.risk)}" />
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <span class="font-medium">{issue.description}</span>
                    <span class="px-2 py-0.5 text-xs rounded-full {getRiskColor(issue.risk)} bg-{issue.risk.toLowerCase()}-500/10">
                      {issue.risk}
                    </span>
                  </div>
                </div>
              </button>

              <!-- Expanded Content -->
              {#if expandedIssue === index}
                <div class="px-4 pb-4" transition:slide|local>
                  <!-- Evidence -->
                  <div class="mb-3 p-3 rounded-lg bg-red-500/10 border border-red-900 text-red-200">
                    <div class="flex gap-2 items-center mb-2 text-sm font-semibold">
                      <Info class="w-4 h-4" />
                      Current Configuration
                    </div>
                    <div class="font-mono text-sm whitespace-pre-wrap">
                      {issue.evidence}
                    </div>
                  </div>

                  <!-- Mitigation -->
                  <div class="p-3 rounded-lg bg-blue-500/10 border border-blue-900 text-blue-200">
                    <div class="flex gap-2 items-center mb-2 text-sm font-semibold">
                      <Shield class="w-4 h-4" />
                      Recommended Action
                    </div>
                    <div class="flex gap-2">
                      <ArrowRight class="w-4 h-4 mt-1 flex-shrink-0" />
                      <span>{issue.mitigation}</span>
                    </div>
                  </div>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Passed Checks -->
    {#if results.headers.passed.length > 0}
      <div>
        <h3 class="text-lg font-semibold flex items-center gap-2 text-green-400 mb-3">
          <Shield class="w-5 h-5" />
          Passed Checks
        </h3>
        <div class="space-y-2">
          {#each results.headers.passed as check}
            <div class="flex items-center gap-2 text-muted-foreground">
              <div class="w-1.5 h-1.5 rounded-full bg-green-500" />
              <span>{check}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
{/if}