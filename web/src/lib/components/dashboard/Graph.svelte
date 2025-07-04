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
      size: 14,
      font: {
        size: 14,
        face: 'Helvetica',
        multi: true,
        bold: {
          color: '#22C55D',
          size: 14,
          face: 'Helvetica'
        }
      },
      borderWidth: 2,
      color: {
        border: '#2B7CE9',
        background: 'rgba(210, 229, 255, 0.2)',
        highlight: {
          border: '#2B7CE9',
          background: '#D2E5FF'
        },
      }
    },
    groups: {
      domain: {
        color: { background: 'rgba(151, 194, 252, 0.9)', border: '#2B7CE9' },
        shape: 'diamond',
        size: 24,
        font: { size: 16, color: '#22C55D' }
      },
      subdomain: {
        color: { background: 'rgba(255, 215, 0, 0.7)', border: '#FFD700' },
        shape: 'dot',
        size: 18,
        font: { size: 14, color: '#22C55D' }
      },
      site: {
        color: { background: 'rgba(210, 229, 255, 0.7)', border: '#2B7CE9' },
        shape: 'dot',
        size: 16,
        font: { size: 14, color: '#22C55D' }
      },
      error: {
        color: { background: 'rgba(255, 153, 153, 0.7)', border: '#CC3333' },
        shape: 'triangle',
        size: 16,
        font: { size: 14, color: '#22C55D' }
      }
    },
    edges: {
      width: 1.5,
      color: { color: 'rgba(120, 120, 120, 0.7)' },
      smooth: {
        type: 'continuous',
        forceDirection: 'none',
        roundness: 0.5
      },
      shadow: {
        enabled: true,
        color: 'rgba(0,0,0,0.2)',
        size: 3,
        x: 1,
        y: 1
      }
    },
    physics: {
      enabled: true,
      solver: 'forceAtlas2Based',
      forceAtlas2Based: {
        gravitationalConstant: -50,
        centralGravity: 0.007,
        springLength: 75,
        springConstant: 0.08,
        damping: 0.4,
        avoidOverlap: 0.2
      },
      stabilization: {
        enabled: true,
        iterations: 1000,
        updateInterval: 25,
        fit: true
      },
      maxVelocity: 50,
      minVelocity: 0.1
    },
    interaction: {
      hover: true,
      tooltipDelay: 200,
      multiselect: false,
      navigationButtons: false,
      keyboard: {
        enabled: true,
        bindToWindow: false
      },
      zoomView: true,
      dragNodes: true,
      dragView: true,
      hideEdgesOnDrag: false,
      hideEdgesOnZoom: false,
      hoverConnectedEdges: true,
      selectable: true,
      selectConnectedEdges: true,
      tooltipDelay: 300
    }
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

<div bind:this={container} class="w-full h-[80vh] rounded bg-transparent"></div>
<style>
:global(.vis-network:focus) {
  outline: none;
  border: none;
}
:global(.vis-network) {
  width: 100% !important;
  height: 100% !important;
  border:none;
  background: transparent !important;
}

:global(.vis-network canvas) {
  background: transparent !important;
}

:global(.vis-tooltip) {
  position: absolute;
  visibility: hidden;
  padding: 12px;
  white-space: nowrap;
  font-family: 'Helvetica', sans-serif;
  font-size: 14px;
  color: #333;
  background-color: rgba(255, 255, 255, 0.9);
  border-radius: 8px;
  border: 1px solid rgba(200, 200, 200, 0.8);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
  z-index: 10000;
  pointer-events: none;
  max-width: 300px;
  transition: opacity 0.3s ease;
}

:global(.tooltip) {
  padding: 2px;
}

:global(.tooltip strong) {
  margin-bottom: 4px;
  display: block;
  font-weight: bold;
}
</style>
