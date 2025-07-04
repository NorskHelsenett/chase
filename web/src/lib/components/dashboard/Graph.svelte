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
          color: '#ff9d4f', // Matches primary orange theme
          size: 14,
          face: 'Helvetica'
        },
        color: '#e6e6e6' // Light text for dark background
      },
      borderWidth: 2,
      color: {
        border: '#e65c00', // Using the gradient color from CSS
        background: 'rgba(38, 38, 38, 0.7)', // Darker background matching site theme
        highlight: {
          border: '#ff8c38', // Brighter version of primary
          background: '#404040' // Slightly lighter than background
        },
      }
    },
    groups: {
      domain: {
        color: { background: 'rgba(230, 92, 0, 0.8)', border: '#ff7b1f' }, // Orange based on site theme
        shape: 'diamond',
        size: 24,
        font: { size: 16, color: '#ffffff' }
      },
      subdomain: {
        color: { background: 'rgba(255, 158, 79, 0.7)', border: '#e65c00' }, // Lighter orange
        shape: 'dot',
        size: 18,
        font: { size: 14, color: '#ffffff' }
      },
      site: {
        color: { background: 'rgba(34, 197, 93, 0.6)', border: '#22C55D' }, // Green color from the theme
        shape: 'dot',
        size: 16,
        font: { size: 14, color: '#ffffff' }
      },
      error: {
        color: { background: 'rgba(255, 76, 76, 0.7)', border: '#ff4c4c' }, // Using alert color from CSS
        shape: 'triangle',
        size: 16,
        font: { size: 14, color: '#ffffff' }
      }
    },
    edges: {
      width: 1.5,
      color: { color: 'rgba(100, 100, 100, 0.7)' }, // Slightly darker for better contrast
      smooth: {
        type: 'continuous',
        forceDirection: 'none',
        roundness: 0.5
      },
      shadow: {
        enabled: true,
        color: 'rgba(230, 92, 0, 0.2)', // Slight orange tint matching primary color
        size: 3,
        x: 1,
        y: 1
      },
      hoverWidth: 2, // Slightly wider on hover
      selectionWidth: 2.5 // Wider when selected
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
  color: #e6e6e6;
  background-color: rgba(32, 32, 32, 0.95);
  border-radius: 8px;
  border: 1px solid rgba(77, 76, 76, 0.5);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
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
  color: #ff9d4f; /* Match primary orange theme */
}
</style>
