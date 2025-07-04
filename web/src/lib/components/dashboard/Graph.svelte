<script lang="ts">
import { onMount } from 'svelte';
import { get } from 'svelte/store';
import type { Writable } from 'svelte/store';

export let graphData: Writable<{ nodes: any[]; edges: any[] }>;
let container: HTMLDivElement;
let network: any;

onMount(async () => {
  const vis = await import('vis-network/standalone');
  const { nodes, edges } = get(graphData);
  const data = { nodes: new vis.DataSet(nodes), edges: new vis.DataSet(edges) };
  const options = {
    layout: { hierarchical: false },
    nodes: {
      shape: 'dot',
      size: 12,
      font: { size: 14 },
      borderWidth: 1
    },
    edges: {
      width: 1,
      color: { color: '#aaa' }
    },
    physics: {
      enabled: true,
      barnesHut: { gravitationalConstant: -30000, springLength: 120, springConstant: 0.04 },
      stabilization: { iterations: 200 }
    },
    interaction: { hover: true, tooltipDelay: 200 }
  };
  network = new vis.Network(container, data, options);
});
</script>

<div bind:this={container} class="w-full h-[70vh] border rounded bg-white"></div>
<style>
:global(.vis-network) {
  width: 100% !important;
  height: 100% !important;
}
</style>
