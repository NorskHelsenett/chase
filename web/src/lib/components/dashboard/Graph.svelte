<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { get } from 'svelte/store';
	import type { Writable } from 'svelte/store';

	type GraphNode = {
		id: string | number;
		label?: string;
		group?: string;
		isDown?: boolean; // Added to track down status
		[key: string]: any; // Additional properties
	};

	type GraphEdge = {
		from: string | number;
		to: string | number;
		[key: string]: any; // Additional properties
	};

	export let graphData: Writable<{ nodes: GraphNode[]; edges: GraphEdge[] }>;
	let container: HTMLDivElement;
	let network: any;

// Cleanup / handler refs that must be available to lifecycle hooks at initialization
let _dragRafId: any = null;
let _draggingState: any = null;
let stabilizationHandler: any = null;

	const STABILIZATION_DELAY_MS = 1000; // Delay to allow final node positioning after stabilization

	onMount(async () => {
		const vis: any = await import('vis-network/standalone');

		try {
			const { nodes, edges } = get(graphData);
			console.log(`Building graph with ${nodes.length} nodes and ${edges.length} edges`);

			// Create new DataSets with uniqueness check
			const nodeDataSet: any = new vis.DataSet();
			const edgeDataSet: any = new vis.DataSet();

			// Add nodes one by one to handle potential duplicates
			nodes.forEach((node) => {
				try {
					if (!nodeDataSet.get(node.id)) {
						// If the node is marked as down, set its group to 'error'
						if (node.isDown === true) {
							console.log(`Node ${node.id} is down, rendering as error`);
							node.group = 'error';
						}

						// Add the node to the dataset
						const { isDown, ...nodeToAdd } = node; // Remove isDown property as vis-network doesn't need it
						nodeDataSet.add(nodeToAdd);
					} else {
						console.log(`Skipping duplicate node: ${node.id}`);
					}
				} catch (e) {
					console.error(`Error adding node ${node.id}:`, e);
				}
			});

			// Add edges one by one
			edges.forEach((edge) => {
				try {
					edgeDataSet.add(edge);
				} catch (e) {
					console.error(`Error adding edge from ${edge.from} to ${edge.to}:`, e);
				}
			});

			const data = { nodes: nodeDataSet, edges: edgeDataSet };
			const options = {
				layout: { hierarchical: false, improvedLayout: false },
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
						}
					}
				},
				groups: {
					domain: {
						color: { background: 'rgba(230, 92, 0, 0.8)', border: '#ff7b1f' }, // Orange based on site theme
						shape: 'diamond',
						size: 36,
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
					// Default neutral color, with explicit hover/highlight set to a pleasant blue
					color: {
						color: 'rgba(100, 100, 100, 0.7)', // Slightly darker for better contrast
						hover: '#60A5FA', // Tailwind sky-400 / pleasant blue on hover
						highlight: '#3B82F6' // Tailwind blue-500 when selected
					},
					smooth: {
						type: 'continuous',
						forceDirection: 'none',
						roundness: 0.5
					},
					shadow: {
						enabled: true,
						// Subtle blue-tinted shadow to match hover color (low alpha)
						color: 'rgba(59, 130, 246, 0.12)',
						size: 4,
						x: 0,
						y: 1
					},
					hoverWidth: 2, // Slightly wider on hover
					selectionWidth: 2.5 // Wider when selected
				},
				physics: {
					enabled: true,
					solver: 'forceAtlas2Based',
					forceAtlas2Based: {
						gravitationalConstant: -150,
						centralGravity: 0.007,
						springLength: 95,
						springConstant: 0.08,
						damping: 0.4,
						avoidOverlap: 0.5
					},
					stabilization: {
						enabled: true,
						iterations: 1000,
						updateInterval: 25,
						fit: true
					},
					maxVelocity: 10, // Reduced from 50
					minVelocity: 0.5, // Increased from 0.1 to make it stop sooner
					timestep: 0.5 // Added timestep to slow down simulation
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
					selectConnectedEdges: true
				}
			};
			network = new vis.Network(container, data as any, options);

			// Add click handler for node interaction
			network.on('click', function (params: any) {
				if (params.nodes.length > 0) {
					const nodeId = params.nodes[0];
					const clickedNode: any = nodeDataSet.get(nodeId);

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

				// Keep a map of original node colors so we can restore them after hover
				const originalNodeColors = new Map();

				function saveOriginalAndSetNodeColor(id: string | number, colorObj: any) {
					try {
						const node = nodeDataSet.get(id);
						const orig = node && node.color ? node.color : null;
						if (!originalNodeColors.has(id)) originalNodeColors.set(id, orig);
						nodeDataSet.update({ id, color: colorObj });
					} catch (e) {
						console.error('Error tinting node', id, e);
					}
				}

				const hoverNodeHandler = function (params: any) {
					if (!params || !params.node) return;
					const nodeId = params.node;
					// Determine parent nodes (edges that point to the hovered node)
					const parentSet = new Set<any>();
					try {
						edgeDataSet.get().forEach((e: any) => {
							if (e && e.to === nodeId) parentSet.add(e.from);
						});
					} catch (e) {
						// Fallback: if edgeDataSet.get() fails, leave parentSet empty
						console.error('Error computing parent set for hover', e);
					}

					// BFS up to depth 4 (first..fourth degree), excluding parents and the hovered node
					const depths: Array<Array<any>> = [];
					const visited = new Set<any>();
					visited.add(nodeId);

					// start with first-degree neighbors
					let current = (network.getConnectedNodes(nodeId) || []).filter((n: any) => n !== nodeId && !parentSet.has(n));
					for (let d = 1; d <= 4; d++) {
						const next: any[] = [];
						const uniqueCurrent = Array.from(new Set(current)).filter((n) => !visited.has(n) && !parentSet.has(n));
						if (uniqueCurrent.length === 0) break;
						depths[d] = uniqueCurrent;
						uniqueCurrent.forEach((n) => visited.add(n));
						// gather neighbors for next depth
						uniqueCurrent.forEach((n) => {
							const cn = network.getConnectedNodes(n) || [];
							cn.forEach((nn: any) => {
								if (!visited.has(nn) && nn !== nodeId && !parentSet.has(nn)) next.push(nn);
							});
						});
						current = next;
					}

					// Apply tint per depth with progressively subtler blues
					// Depth 1: strongest
					const depth1 = depths[1] || [];
					depth1.forEach((id: any) => {
						saveOriginalAndSetNodeColor(id, {
							border: '#3B82F6',
							background: 'rgba(59,130,246,0.14)',
							highlight: { border: '#3B82F6', background: 'rgba(59,130,246,0.18)' }
						});
					});

					// Depth 2: slightly less strong
					const depth2 = depths[2] || [];
					depth2.forEach((id: any) => {
						saveOriginalAndSetNodeColor(id, {
							border: '#60A5FA',
							background: 'rgba(96,165,250,0.09)',
							highlight: { border: '#60A5FA', background: 'rgba(96,165,250,0.12)' }
						});
					});

					// Depth 3: subtle
					const depth3 = depths[3] || [];
					depth3.forEach((id: any) => {
						saveOriginalAndSetNodeColor(id, {
							border: '#93C5FD',
							background: 'rgba(147,197,253,0.06)',
							highlight: { border: '#93C5FD', background: 'rgba(147,197,253,0.09)' }
						});
					});

					// Depth 4: very subtle
					const depth4 = depths[4] || [];
					depth4.forEach((id: any) => {
						saveOriginalAndSetNodeColor(id, {
							border: '#BFDBFE',
							background: 'rgba(191,219,254,0.04)',
							highlight: { border: '#BFDBFE', background: 'rgba(191,219,254,0.06)' }
						});
					});
				};

				const blurNodeHandler = function () {
					// Restore original colors
					originalNodeColors.forEach((orig, id) => {
						try {
							if (orig) nodeDataSet.update({ id, color: orig });
							else nodeDataSet.update({ id, color: null });
						} catch (e) {
							console.error('Error restoring node color', id, e);
						}
					});
					originalNodeColors.clear();
				};

				network.on('hoverNode', hoverNodeHandler);
				network.on('blurNode', blurNodeHandler);

				// Dragging: move children (and their descendants) together with the dragged node
				// Use top-level _dragRafId and _draggingState declared above (avoid shadowing)

				network.on('dragStart', (params: any) => {
					try {
						if (!params || !params.nodes || params.nodes.length === 0) return;
						// For now, use the first dragged node
						const dragged = params.nodes[0];
						// Build descendant set following outgoing edges (from -> to)
						const descendants = new Set<any>();
						const queue: any[] = [dragged];
						while (queue.length) {
							const cur = queue.shift();
							edgeDataSet.get().forEach((e: any) => {
								if (e && e.from === cur && e.to !== dragged && !descendants.has(e.to)) {
									descendants.add(e.to);
									queue.push(e.to);
								}
							});
						}

						if (descendants.size === 0) return; // nothing to move

						// Record original positions
						const descArr = Array.from(descendants);
						const origPositions = network.getPositions(descArr);
						const draggedPosObj = network.getPositions([dragged]);
						const origDraggedPos = draggedPosObj && draggedPosObj[dragged] ? draggedPosObj[dragged] : null;
						if (!origDraggedPos) return;

						_draggingState = { dragged, descArr, origPositions, origDraggedPos };

						// Use requestAnimationFrame for smoother movement
						function dragFrame() {
							if (!_draggingState) return;
							const posNowObj = network.getPositions([_draggingState.dragged]);
							const posNow = posNowObj && posNowObj[_draggingState.dragged] ? posNowObj[_draggingState.dragged] : null;
							if (!posNow) {
								_dragRafId = requestAnimationFrame(dragFrame);
								return;
							}
							const dx = posNow.x - _draggingState.origDraggedPos.x;
							const dy = posNow.y - _draggingState.origDraggedPos.y;
							_draggingState.descArr.forEach((id: any) => {
								const op = _draggingState.origPositions[id];
								if (op) {
									try {
										network.moveNode(id, op.x + dx, op.y + dy);
									} catch (e) {
										// ignore individual move errors
									}
								}
							});
							_dragRafId = requestAnimationFrame(dragFrame);
						}
						_dragRafId = requestAnimationFrame(dragFrame);
					} catch (e) {
						console.error('Error initializing drag move of children', e);
					}
				});

				network.on('dragEnd', () => {
					if (_dragRafId) {
						cancelAnimationFrame(_dragRafId);
						_dragRafId = null;
					}
					_draggingState = null;
				});

			// Keep physics enabled but make the network static after stabilization
			stabilizationHandler = function () {
				setTimeout(() => {
					// Keep physics enabled but with minimal movement by adjusting parameters
					network.setOptions({
						physics: {
							enabled: true,
							solver: 'forceAtlas2Based',
							forceAtlas2Based: {
								gravitationalConstant: -150,
								centralGravity: 0.007,
								springLength: 95,
								springConstant: 0.08,
								damping: 0.9,
								avoidOverlap: 0.5
							},
							minVelocity: 0.75, // Higher value to stop movement sooner
							maxVelocity: 0.75, // Limit maximum velocity
							timestep: 0.25, // Slower updates
							stabilization: {
								enabled: false // Disable further stabilization
							}
						}
					});
					console.log('Network stabilized - static positioning enabled');
				}, STABILIZATION_DELAY_MS); // Short delay to allow final node positioning
			};
			network.on('stabilizationIterationsDone', stabilizationHandler as any);

			// NOTE: cleanup is registered at component initialization below using onDestroy
		} catch (error) {
			console.error('Error initializing graph:', error);
		}
	});

	// Register component-level cleanup during initialization (allowed by Svelte)
	onDestroy(() => {
		try {
			if (network) {
				// If we have a specific stabilization handler, remove that one, otherwise remove all
				if (stabilizationHandler) network.off('stabilizationIterationsDone', stabilizationHandler);
				else network.off('stabilizationIterationsDone');

				// Remove other handlers (removeAll variants are fine)
				network.off('hoverNode');
				network.off('blurNode');
				network.off('dragStart');
				network.off('dragEnd');
			}
		} catch (e) {
			// swallow cleanup errors
		}

		if (_dragRafId) {
			try {
				cancelAnimationFrame(_dragRafId);
			} catch (e) {
				// ignore
			}
			_dragRafId = null;
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
		border: none;
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
