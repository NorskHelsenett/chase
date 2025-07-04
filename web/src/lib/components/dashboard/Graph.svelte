<script lang="ts">
import { onMount } from 'svelte';
import { get } from 'svelte/store';
import type { Writable } from 'svelte/store';

export let graphData: Writable<{ nodes: any[]; edges: any[] }>;
let container: HTMLDivElement;
let network: any;

onMount(async () => {
  const vis = await import('vis-network/standalone');

  try {
    const { nodes, edges } = get(graphData);
    console.log(`Building graph with ${nodes.length} nodes and ${edges.length} edges`);

    // Create new DataSets with uniqueness check
    const nodeDataSet = new vis.DataSet();
    const edgeDataSet = new vis.DataSet();

    // Add nodes one by one to handle potential duplicates
    nodes.forEach(node => {
      try {
        if (!nodeDataSet.get(node.id)) {
          nodeDataSet.add(node);
        } else {
          console.log(`Skipping duplicate node: ${node.id}`);
        }
      } catch (e) {
        console.error(`Error adding node ${node.id}:`, e);
      }
    });

    // Add edges one by one
    edges.forEach(edge => {
      try {
        edgeDataSet.add(edge);
      } catch (e) {
        console.error(`Error adding edge from ${edge.from} to ${edge.to}:`, e);
      }
    });

    const data = { nodes: nodeDataSet, edges: edgeDataSet };
    const options = {
    layout: { hierarchical: false },
    nodes: {
      shape: 'dot',
      size: 12,
      font: { size: 14 },
      borderWidth: 1,
      color: {
        border: '#2B7CE9',
        background: '#D2E5FF',
        highlight: {
          border: '#2B7CE9',
          background: '#D2E5FF'
        },
      }
    },
    groups: {
      domain: {
        color: { background: '#97C2FC', border: '#2B7CE9' },
        shape: 'diamond',
        size: 16
      },
      subdomain: {
        color: { background: '#FFFF00', border: '#FFD700' },
        shape: 'dot',
        size: 14
      },
      site: {
        color: { background: '#D2E5FF', border: '#2B7CE9' },
        shape: 'dot',
        size: 12
      },
      error: {
        color: { background: '#FF9999', border: '#CC3333' },
        shape: 'triangle',
        size: 12
      }
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

  // Add click handler for node interaction
  network.on('click', function(params) {
    if (params.nodes.length > 0) {
      const nodeId = params.nodes[0];
      const clickedNode = nodeDataSet.get(nodeId);

      // If it's a site node, try to open the URL
      if (clickedNode && clickedNode.group === 'site') {
        const url = clickedNode.label;
        if (url) {
          const fullUrl = url.startsWith('http') ? url : `https://${url}`;
          window.open(fullUrl, '_blank');
        }
      }
    }
  });
  } catch (error) {
    console.error("Error initializing graph:", error);
  }
});
</script>

<div bind:this={container} class="w-full h-[90vh] border rounded bg-white"></div>
<style>
:global(.vis-network) {
  width: 100% !important;
  height: 100% !important;
}
</style>
