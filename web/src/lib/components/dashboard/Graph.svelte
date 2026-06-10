<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { get } from 'svelte/store';
	import type { Writable } from 'svelte/store';

	type GraphNode = {
		id: string | number;
		label?: string;
		group?: string;
		isDown?: boolean;
		[k: string]: any;
	};
	type GraphEdge = { from: string | number; to: string | number; [k: string]: any };


	let container: HTMLDivElement = $state();
	let network: any;
	let nodeDataSet: any;
	let edgeDataSet: any;
	let unsubscribe: () => void;
	interface Props {
		graphData: Writable<{ nodes: GraphNode[]; edges: GraphEdge[] }>;
		loading?: boolean;
	}

	let { graphData, loading = $bindable(true) }: Props = $props();
	let loadPct = $state(0);

	// --- helpers ---
	const edgeId = (e: GraphEdge) => `${e.from}|${e.to}`;

	// Tune these if you want tighter/looser clusters
	const LEAF_EDGE_LEN = 110; // domain hub -> server
	const HUB_EDGE_LEN = 220; // domain hub -> sub-domain hub

	const GOLDEN_ANGLE = 2.399963229728653;

	// Keep track of current highlighting so we can reset colors on the next click
	let lastHighlightedNodes: Array<string | number> = [];
	let lastHighlightedEdges: Array<string> = [];

	// Drag state so children move together with the dragged parent
	let dragState: {
		anchorId: string | number | null;
		startPos: Record<string | number, { x: number; y: number }>;
		childIds: Array<string | number>;
	} = {
		anchorId: null,
		startPos: {},
		childIds: []
	};

	// --- styling defaults (used only for UNgrouped nodes) ---
	const DEFAULT_NODE_COLOR = {
		border: '#e65c00',
		background: 'rgba(38,38,38,0.7)',
		highlight: { border: '#ff8c38', background: '#404040' }
	};
	const DEFAULT_NODE_FONT = {
		size: 14,
		face: 'Helvetica',
		multi: true,
		bold: { color: '#ff9d4f', size: 14, face: 'Helvetica' },
		color: '#e6e6e6'
	};

	function resetNodeToBase(n: any) {
		// Keep group styling by removing custom color if grouped; use defaults otherwise
		const ret: any = { ...n, borderWidth: 2, font: n.font ?? DEFAULT_NODE_FONT };
		if (n.group) {
			delete ret.color; // let groups.{...}.color take over
		} else {
			ret.color = DEFAULT_NODE_COLOR;
		}
		return ret;
	}

	function prepareData(nodes: GraphNode[], edges: GraphEdge[]) {
		// Unique nodes, map isDown -> error
		const seen = new Set<string | number>();
		const uniqueNodes: any[] = [];
		const groupOf: Map<string | number, string | undefined> = new Map();
		for (const n of nodes) {
			if (seen.has(n.id)) continue;
			seen.add(n.id);
			const { isDown, ...rest } = n as any;
			if (isDown === true) (rest as any).group = 'error';
			const tip = buildTooltip(rest);
			if (tip) rest.title = tip;
			uniqueNodes.push(rest);
			groupOf.set(rest.id, rest.group);
		}

		// Edge ids are from|to so unchanged edges survive updates; hub-to-hub
		// edges are longer than hub-to-leaf so subtrees fan out inside a cluster.
		const seenEdges = new Set<string>();
		const processedEdges: any[] = [];
		for (const e of edges) {
			const id = edgeId(e);
			if (seenEdges.has(id)) continue;
			seenEdges.add(id);
			const bothHubs = groupOf.get(e.from) === 'domain' && groupOf.get(e.to) === 'domain';
			processedEdges.push({
				...e,
				id,
				length: bothHubs ? HUB_EDGE_LEN : LEAF_EDGE_LEN
			});
		}

		return { uniqueNodes, processedEdges };
	}

	const clusterKey = (n: any) => n?.cluster || String(n?.id ?? '');

	// vis restarts the simulation on every DataSet.update (clicks, filters,
	// highlights). The interactive physics below is low-energy and self-stops
	// via minVelocity, but this watchdog guarantees it can never run away.
	let calmTimer: ReturnType<typeof setTimeout> | undefined;
	function calmDown(ms = 3000) {
		clearTimeout(calmTimer);
		calmTimer = setTimeout(() => {
			try {
				network?.stopSimulation?.();
			} catch {}
		}, ms);
	}

	// --- tooltip card (screenshot + status box) ---
	const tooltipImages = new Map<string | number, HTMLImageElement>();

	function buildTooltip(n: any): HTMLElement | undefined {
		const meta = n.meta;
		if (!meta) return undefined;

		const tip = document.createElement('div');
		tip.className = 'graph-tip';

		const info = document.createElement('div');
		info.className = 'graph-tip-info';

		const title = document.createElement('div');
		title.className = 'graph-tip-title';
		title.textContent = meta.kind === 'domain' ? meta.domain : meta.url;
		info.appendChild(title);

		const row = document.createElement('div');
		row.className = 'graph-tip-row';

		if (meta.kind === 'domain') {
			row.textContent = `${meta.count} monitored site${meta.count === 1 ? '' : 's'}`;
			info.appendChild(row);
		} else if (meta.kind === 'invalid') {
			row.textContent = 'Invalid URL format';
			info.appendChild(row);
		} else {
			const up = meta.status === 'up';
			const dot = document.createElement('span');
			dot.className = `graph-tip-dot ${up ? 'up' : 'down'}`;
			row.appendChild(dot);
			const bits = [up ? 'Up' : 'Down'];
			if (typeof meta.responseTimeMs === 'number') bits.push(`${Math.round(meta.responseTimeMs)} ms`);
			if (meta.statusCode) bits.push(`HTTP ${meta.statusCode}`);
			row.appendChild(document.createTextNode(bits.join(' · ')));
			info.appendChild(row);

			if (!up && meta.error) {
				const err = document.createElement('div');
				err.className = 'graph-tip-note';
				err.textContent = meta.error;
				info.appendChild(err);
			}

			// Screenshot loads lazily: src is set on first hover (hoverNode),
			// so building the graph doesn't fire a request per node
			tip.classList.add('has-shot');
			const img = document.createElement('img');
			img.className = 'graph-tip-shot';
			img.alt = '';
			const cleanUrl = String(meta.url)
				.replace(/^(https?:\/\/)/, '')
				.replace(/\/$/, '');
			img.dataset.src = `/api/screenshot/${cleanUrl}?cached=true&thumb=true`;
			img.onerror = () => {
				img.remove();
				tip.classList.remove('has-shot');
			};
			tip.appendChild(img);
			tooltipImages.set(n.id, img);
		}

		if (meta.note) {
			const note = document.createElement('div');
			note.className = 'graph-tip-note';
			note.textContent = `Note: ${meta.note}`;
			info.appendChild(note);
		}

		tip.appendChild(info);
		return tip;
	}

	// Place each cluster (root domain) on a golden-angle spiral so they start
	// scattered instead of piled on the origin; physics then only needs to
	// tidy nodes locally within each cluster.
	function seedClusterPositions(uniqueNodes: any[]) {
		const clusters = new Map<string, any[]>();
		for (const n of uniqueNodes) {
			const key = clusterKey(n);
			const members = clusters.get(key) || [];
			members.push(n);
			clusters.set(key, members);
		}

		// Biggest clusters near the center, small ones further out
		const entries = [...clusters.entries()].sort((a, b) => b[1].length - a[1].length);
		const radii = entries.map(([, members]) => 110 + 55 * Math.sqrt(members.length));
		const avgRadius = radii.reduce((sum, r) => sum + r, 0) / (radii.length || 1);
		const spacing = Math.max(300, 2.4 * avgRadius);

		entries.forEach(([, members], i) => {
			const angle = i * GOLDEN_ANGLE;
			const dist = spacing * Math.sqrt(i);
			const cx = Math.cos(angle) * dist;
			const cy = Math.sin(angle) * dist;
			// Hubs first so the domain diamond starts at the cluster center
			members.sort((a, b) => (a.group === 'domain' ? -1 : 1) - (b.group === 'domain' ? -1 : 1));
			members.forEach((n, j) => {
				if (n.x !== undefined && n.y !== undefined) return;
				const a = j * GOLDEN_ANGLE;
				const d = radii[i] * Math.sqrt(j / members.length);
				n.x = cx + Math.cos(a) * d;
				n.y = cy + Math.sin(a) * d;
			});
		});
	}

	function updateData(nodes: GraphNode[], edges: GraphEdge[]) {
		if (!network || !nodeDataSet || !edgeDataSet) return;

		const { uniqueNodes, processedEdges } = prepareData(nodes, edges);

		// Preserve positions for nodes that remain
		const existingIds = nodeDataSet.getIds();
		const existingPositions = existingIds.length ? network.getPositions(existingIds) : {};
		const nextIds = new Set(uniqueNodes.map((n) => String(n.id)));

		// Cluster centroids of surviving nodes, so newly added nodes drop near
		// their own cluster and the low-energy physics only has to tidy locally
		const centroids = new Map<string, { x: number; y: number; count: number }>();
		let maxDist = 0;
		for (const id of existingIds) {
			const pos = (existingPositions as any)[id];
			if (!pos) continue;
			maxDist = Math.max(maxDist, Math.hypot(pos.x, pos.y));
			const key = clusterKey(nodeDataSet.get(id));
			const c = centroids.get(key) || { x: 0, y: 0, count: 0 };
			c.x += pos.x;
			c.y += pos.y;
			c.count++;
			centroids.set(key, c);
		}

		// Remove nodes that disappeared (filter/update fix)
		const toRemoveNodes = existingIds.filter((id: any) => !nextIds.has(String(id)));
		if (toRemoveNodes.length) nodeDataSet.remove(toRemoveNodes);

		// Add or update nodes, injecting previous x/y when available
		const upserts = uniqueNodes.map((n, i) => {
			const pos = (existingPositions as any)[n.id];
			if (pos) return { ...n, x: pos.x, y: pos.y };
			const c = centroids.get(clusterKey(n));
			const a = i * GOLDEN_ANGLE;
			if (c?.count) {
				// Near the cluster's centroid, slightly offset to avoid stacking
				return { ...n, x: c.x / c.count + Math.cos(a) * 60, y: c.y / c.count + Math.sin(a) * 60 };
			}
			// Brand-new cluster: place on a ring outside the current layout
			const ring = maxDist + 300;
			return { ...n, x: Math.cos(a) * ring, y: Math.sin(a) * ring };
		});
		if (upserts.length) nodeDataSet.update(upserts);

		// Edges: remove missing, then upsert
		const existingEdgeIds = edgeDataSet.getIds();
		const nextEdgeIdSet = new Set(processedEdges.map((e) => String(e.id)));
		const toRemoveEdges = existingEdgeIds.filter((id: any) => !nextEdgeIdSet.has(String(id)));
		if (toRemoveEdges.length) edgeDataSet.remove(toRemoveEdges);
		if (processedEdges.length) edgeDataSet.update(processedEdges);

		// The dataset updates restart the low-energy simulation; make sure it rests
		calmDown();
		network.redraw();
	}

	// --- highlight helpers (now does ALL descendants) ---
	function resetHighlights() {
		if (!nodeDataSet || !edgeDataSet) return;

		// Store positions to prevent movement
		network.storePositions();

		if (lastHighlightedNodes.length) {
			const resetNodes = lastHighlightedNodes
				.map((id) => nodeDataSet.get(id))
				.filter(Boolean)
				.map((n: any) => {
					const pos = network.getPositions([n.id])[n.id];
					return { ...resetNodeToBase(n), x: pos?.x, y: pos?.y };
				});
			if (resetNodes.length) nodeDataSet.update(resetNodes);
			lastHighlightedNodes = [];
		}

		if (lastHighlightedEdges.length) {
			const resetEdges = lastHighlightedEdges
				.map((id) => edgeDataSet.get(id))
				.filter(Boolean)
				.map((e: any) => ({
					...e,
					color: { color: 'rgba(100,100,100,0.7)', highlight: '#3B82F6', hover: '#60A5FA' },
					width: 1.5
				}));
			if (resetEdges.length) edgeDataSet.update(resetEdges);
			lastHighlightedEdges = [];
		}
	}

	function getDescendantsAndEdges(rootId: string | number) {
		// BFS over directed edges: follow e.from -> e.to
		const allEdges: any[] = edgeDataSet.get();
		const outMap = new Map<string | number, Array<any>>();
		for (const e of allEdges) {
			const key = String(e.from);
			const arr = outMap.get(key) || [];
			arr.push(e);
			outMap.set(key, arr);
		}

		const visited = new Set<string | number>();
		const q: Array<string | number> = [rootId];
		const edgesTouched: Array<any> = [];

		visited.add(rootId);

		while (q.length) {
			const cur = q.shift()!;
			const outgoing = outMap.get(String(cur)) || [];
			for (const e of outgoing) {
				edgesTouched.push(e);
				if (!visited.has(e.to)) {
					visited.add(e.to);
					q.push(e.to);
				}
			}
		}

		// Remove the root itself from node highlighting set; we only color descendants
		visited.delete(rootId);

		return {
			nodeIds: Array.from(visited),
			edgeIds: edgesTouched.map((e) => e.id),
			edges: edgesTouched
		};
	}

	function highlightSubtree(parentId: string | number) {
		resetHighlights();

		const { nodeIds, edges, edgeIds } = getDescendantsAndEdges(parentId);
		const blue = '#3B82F6';

		// Store positions to prevent movement
		network.storePositions();

		// Edges first
		const updatedEdges = edges.map((e) => ({
			...e,
			color: { color: blue, highlight: blue, hover: blue },
			width: 2.5
		}));
		if (updatedEdges.length) edgeDataSet.update(updatedEdges);

		// Nodes (descendants only)
		const updatedNodes = nodeIds
			.map((id) => nodeDataSet.get(id))
			.filter(Boolean)
			.map((n: any) => {
				const pos = network.getPositions([n.id])[n.id];
				return {
					...n,
					x: pos?.x,
					y: pos?.y,
					color: {
						...(n.color ?? {}),
						border: blue,
						background: 'rgba(59,130,246,0.15)',
						highlight: { border: blue, background: 'rgba(59,130,246,0.22)' }
					}
				};
			});
		if (updatedNodes.length) nodeDataSet.update(updatedNodes);

		lastHighlightedNodes = nodeIds;
		lastHighlightedEdges = edgeIds;
	}

	let safetyTimer: ReturnType<typeof setTimeout> | undefined;

	onMount(async () => {
		const vis: any = await import('vis-network/standalone');

		const initial = get(graphData);
		const { uniqueNodes, processedEdges } = prepareData(initial.nodes, initial.edges);
		seedClusterPositions(uniqueNodes);

		nodeDataSet = new vis.DataSet(uniqueNodes);
		edgeDataSet = new vis.DataSet(processedEdges);

		const options = {
			layout: { improvedLayout: false, randomSeed: 42 },
			nodes: {
				shape: 'dot',
				size: 14,
				font: {
					size: 14,
					face: 'Helvetica',
					multi: true,
					bold: { color: '#ff9d4f', size: 14, face: 'Helvetica' },
					color: '#e6e6e6'
				},
				borderWidth: 2,
				color: {
					border: '#e65c00',
					background: 'rgba(38,38,38,0.7)',
					highlight: { border: '#ff8c38', background: '#404040' }
				}
			},
			groups: {
				domain: {
					color: { background: 'rgba(230,92,0,0.8)', border: '#ff7b1f' },
					shape: 'diamond',
					size: 36,
					font: { size: 16, color: '#fff' }
				},
				subdomain: {
					color: { background: 'rgba(255,158,79,0.7)', border: '#e65c00' },
					shape: 'dot',
					size: 18,
					font: { size: 14, color: '#fff' }
				},
				site: {
					color: { background: 'rgba(34,197,93,0.6)', border: '#22C55D' },
					shape: 'dot',
					size: 16,
					font: { size: 14, color: '#fff' }
				},
				error: {
					color: { background: 'rgba(255,76,76,0.7)', border: '#ff4c4c' },
					shape: 'triangle',
					size: 16,
					font: { size: 14, color: '#fff' }
				}
			},
			edges: {
				width: 1.5,
				smooth: false,
				shadow: { enabled: false },
				color: { color: 'rgba(100,100,100,0.7)', hover: '#60A5FA', highlight: '#3B82F6' },
				hoverWidth: 2,
				selectionWidth: 2.5
			},
			interaction: {
				hover: true,
				hoverConnectedEdges: true,
				multiselect: false,
				zoomView: true,
				dragNodes: true,
				dragView: true,
				selectable: true
			},
			physics: {
				enabled: true,
				solver: 'barnesHut',
				barnesHut: {
					gravitationalConstant: -8000, // strong repulsion spreads sibling leaves apart
					centralGravity: 0.01, // nearly none, so clusters stay scattered instead of collapsing to one blob
					springLength: 110,
					springConstant: 0.04, // soft springs so repulsion can stretch them
					damping: 0.5,
					avoidOverlap: 0.6
				},
				stabilization: { enabled: true, iterations: 400, updateInterval: 25, fit: true },
				adaptiveTimestep: true
			}
		} as any;

		network = new vis.Network(container, { nodes: nodeDataSet, edges: edgeDataSet }, options);

		// Show rough progress while hidden stabilization runs
		network.on &&
			network.on('stabilizationProgress', (p: any) => {
				if (!p?.total) return;
				loadPct = Math.min(100, Math.round((p.iterations / p.total) * 100));
			});

		// Stabilize once, then hand over to low-energy interactive physics:
		// springy drags and a brief wobble after interactions, but it always
		// comes to rest — high minVelocity stops it on its own and calmDown()
		// is the backstop (the old forceAtlas2 phase had neither and ran forever).
		let finalized = false;
		function finalize() {
			if (finalized) return;
			finalized = true;
			try { network.stopSimulation?.(); } catch {}
			try { network.storePositions(); } catch {}
			try {
				network.setOptions({
					physics: {
						enabled: true,
						solver: 'barnesHut',
						barnesHut: {
							// Keep repulsion close to the stabilization phase so the
							// first interactive wobble doesn't re-compress the layout
							gravitationalConstant: -6000,
							centralGravity: 0.005,
							springLength: 110,
							springConstant: 0.03,
							damping: 0.85,
							avoidOverlap: 0
						},
						stabilization: false,
						maxVelocity: 12,
						minVelocity: 1.2
					}
				});
			} catch {}
			calmDown();
			requestAnimationFrame(() => {
				loading = false;
				try {
					resetHighlights();
					network.unselectAll?.();
					network.fit?.({ animation: false });
				} catch {}
			});
		}

		network.once('stabilizationIterationsDone', finalize);

		// Safety net: force finish if stabilization takes too long on huge graphs
		safetyTimer = setTimeout(finalize, 10000);

		// --- Interactions ---
		// Click: background clears; node click highlights whole subtree and opens site URLs
		network.on('click', (params: any) => {
			calmDown();
			if (!params.nodes?.length) {
				resetHighlights();
				network.unselectAll();
				return;
			}

			const nodeId = params.nodes[0];
			const n = nodeDataSet.get(nodeId);

			highlightSubtree(nodeId);

			// Click-to-open for site nodes
			if (n?.group === 'site' && n?.label) {
				const url = String(n.label);
				const fullUrl = url.startsWith('http') ? url : `https://${url}`;
				window.open(fullUrl, '_blank');
			}
		});

		// Deselect event from vis also clears
		network.on('deselectNode', () => {
			resetHighlights();
			calmDown();
		});

		// Lazily load the tooltip screenshot the first time a node is hovered
		network.on('hoverNode', (params: any) => {
			const img = tooltipImages.get(params.node);
			if (img?.dataset.src && !img.getAttribute('src')) {
				img.src = img.dataset.src;
			}
		});

		// Drag parent and its direct children as a unit (descendants can be large; keep it direct for perf)
		network.on('dragStart', (params: any) => {
			if (!params.nodes?.length) return;
			const anchorId = params.nodes[0];
			const allEdges: any[] = edgeDataSet.get();
			const children = allEdges.filter((e) => String(e.from) === String(anchorId)).map((e) => e.to);
			const ids = [anchorId, ...children];
			const pos = network.getPositions(ids);
			// Delta is measured pointer-to-pointer, not pointer-to-node-center,
			// so children don't jump by the grab offset
			dragState = {
				anchorId,
				startPos: { ...(pos as any), __pointer: params.pointer.canvas },
				childIds: children
			};
		});

		network.on('dragging', (params: any) => {
			if (!dragState.anchorId) return;
			const start = (dragState.startPos as any).__pointer;
			const cur = params.pointer.canvas;
			const dx = cur.x - start.x;
			const dy = cur.y - start.y;
			// Move children by the same delta
			for (const cid of dragState.childIds) {
				const cStart = dragState.startPos[cid];
				if (cStart) network.moveNode(cid, cStart.x + dx, cStart.y + dy);
			}
		});

		network.on('dragEnd', () => {
			dragState = { anchorId: null, startPos: {}, childIds: [] };
			calmDown();
		});

		// Subscribe to reactive updates (filtering, etc.) — skip the immediate
		// first emission, it's the same data the network was constructed with
		let firstEmission = true;
		unsubscribe = graphData.subscribe(({ nodes, edges }) => {
			if (firstEmission) {
				firstEmission = false;
				return;
			}
			updateData(nodes, edges);
		});
	});

	onDestroy(() => {
		clearTimeout(safetyTimer);
		clearTimeout(calmTimer);
		try {
			unsubscribe?.();
		} catch {}
		try {
			network?.off('click');
			network?.off('dragStart');
			network?.off('dragging');
			network?.off('dragEnd');
			network?.off('deselectNode');
		} catch {}
		try {
			network?.destroy();
		} catch {}
		try {
			nodeDataSet?.clear();
			edgeDataSet?.clear();
		} catch {}
	});
</script>

<div class="graph-wrapper">
	<div bind:this={container} class="w-full h-[80vh] rounded bg-transparent"></div>

	{#if loading}
		<div class="graph-loading" role="status" aria-live="polite" aria-label="Loading graph">
			<div class="spinner" aria-hidden="true"></div>
			<div class="label">
				Loading graph{#if loadPct}&nbsp;— {loadPct}%{/if}
			</div>
		</div>
	{/if}
</div>

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
	/* Scoped under .graph-wrapper to out-rank vis-network's injected
	   div.vis-tooltip defaults (white background), which load after us */
	.graph-wrapper :global(div.vis-tooltip) {
		position: absolute;
		visibility: hidden;
		padding: 0;
		white-space: normal;
		font-family: 'Helvetica', sans-serif;
		font-size: 14px;
		color: #e6e6e6;
		background-color: rgba(32, 32, 32, 0.95);
		border-radius: 10px;
		border: 1px solid rgba(77, 76, 76, 0.5);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		z-index: 10000;
		pointer-events: none;
		max-width: 320px;
		overflow: hidden;
		transition: opacity 0.3s ease;
	}
	:global(.graph-tip) {
		min-width: 170px;
		max-width: 280px;
	}
	:global(.graph-tip.has-shot) {
		width: 280px;
	}
	:global(.graph-tip-shot) {
		display: block;
		width: 100%;
		height: 158px;
		object-fit: cover;
		object-position: top;
		background: #161616;
		border-bottom: 1px solid rgba(77, 76, 76, 0.5);
	}
	:global(.graph-tip-info) {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 10px 12px;
	}
	:global(.graph-tip-title) {
		font-weight: bold;
		color: #ff9d4f;
		word-break: break-all;
	}
	:global(.graph-tip-row) {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		color: #cfcfcf;
	}
	:global(.graph-tip-dot) {
		flex: none;
		width: 8px;
		height: 8px;
		border-radius: 9999px;
	}
	:global(.graph-tip-dot.up) {
		background: #22c55d;
	}
	:global(.graph-tip-dot.down) {
		background: #ff4c4c;
	}
	:global(.graph-tip-note) {
		font-size: 12px;
		font-style: italic;
		color: #9ca3af;
		word-break: break-word;
	}
	:global(.tooltip) {
		padding: 2px;
	}
	:global(.tooltip strong) {
		margin-bottom: 4px;
		display: block;
		font-weight: bold;
		color: #ff9d4f;
	}

	/* Graph loading overlay */
	.graph-wrapper {
		position: relative;
	}
	.graph-loading {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
		align-items: center;
		justify-content: center;
		background: linear-gradient(to bottom, rgba(16, 16, 16, 0.6), rgba(16, 16, 16, 0.6));
		backdrop-filter: blur(2px);
		z-index: 20;
	}
	.spinner {
		width: 44px;
		height: 44px;
		border-radius: 9999px;
		border: 4px solid rgba(255, 255, 255, 0.2);
		border-top-color: #22c55d;
		animation: spin 0.9s linear infinite;
	}
	.label {
		font-family: Helvetica, Arial, sans-serif;
		font-size: 14px;
		color: #e6e6e6;
	}
	@media (prefers-reduced-motion: reduce) {
		.spinner {
			animation: none;
			border-top-color: rgba(255, 255, 255, 0.6);
		}
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
