<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import type { Writable } from 'svelte/store';
	import {
		computeGalaxyModel,
		computePositions,
		clusterCenter,
		type GalaxyModel
	} from '$lib/utils/galaxyLayout';

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

	// --- helpers ---
	const edgeId = (e: GraphEdge) => `${e.from}|${e.to}`;

	// Above this node count, leaf labels are hidden (they're unreadable noise
	// anyway and text is by far the most expensive thing to draw per frame).
	// Domain hubs keep their labels; leaves show theirs on hover and highlight.
	const LABEL_LIMIT = 300;
	const HIGHLIGHT_LABEL_LIMIT = 150;
	let labelsVisible = true;

	// Keep track of current highlighting so we can reset colors on the next click
	let lastHighlightedNodes: Array<string | number> = [];
	let lastHighlightedEdges: Array<string> = [];

	// Drag state so descendants move together with the dragged parent
	let dragState: {
		anchorId: string | number | null;
		pointerStart: { x: number; y: number } | null;
		startPos: Record<string, { x: number; y: number }>;
		childIds: Array<string | number>;
	} = { anchorId: null, pointerStart: null, startPos: {}, childIds: [] };

	// --- styling ---
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
	// Explicit per-group colors: also used to truly reset a node after a
	// highlight (DataSet.update can't delete a field, so "remove the color and
	// let the group win" silently kept the old highlight color).
	const GROUP_COLORS: Record<string, any> = {
		domain: {
			background: 'rgba(230,92,0,0.8)',
			border: '#ff7b1f',
			highlight: { border: '#ff8c38', background: 'rgba(230,92,0,0.9)' }
		},
		subdomain: {
			background: 'rgba(255,158,79,0.7)',
			border: '#e65c00',
			highlight: { border: '#ff8c38', background: 'rgba(255,158,79,0.85)' }
		},
		site: {
			background: 'rgba(34,197,93,0.6)',
			border: '#22C55D',
			highlight: { border: '#4ade80', background: 'rgba(34,197,93,0.8)' }
		},
		error: {
			background: 'rgba(255,76,76,0.7)',
			border: '#ff4c4c',
			highlight: { border: '#ff6b6b', background: 'rgba(255,76,76,0.85)' }
		}
	};
	const BASE_EDGE_COLOR = { color: 'rgba(100,100,100,0.7)', hover: '#60A5FA', highlight: '#3B82F6' };

	const baseLabel = (n: any) =>
		labelsVisible || n.group === 'domain' ? (n.fullLabel ?? n.label ?? '') : '';

	function resetNodeToBase(n: any) {
		return {
			...n,
			borderWidth: 2,
			font: n.font ?? DEFAULT_NODE_FONT,
			label: baseLabel(n),
			color: GROUP_COLORS[n.group] ?? DEFAULT_NODE_COLOR
		};
	}

	function prepareData(nodes: GraphNode[], edges: GraphEdge[]) {
		// Unique nodes, map isDown -> error
		const seen = new Set<string | number>();
		const uniqueNodes: any[] = [];
		for (const n of nodes) {
			if (seen.has(n.id)) continue;
			seen.add(n.id);
			const { isDown, ...rest } = n as any;
			if (isDown === true) (rest as any).group = 'error';
			const tip = buildTooltip(rest);
			if (tip) rest.title = tip;
			rest.fullLabel = rest.label;
			uniqueNodes.push(rest);
		}
		labelsVisible = uniqueNodes.length <= LABEL_LIMIT;
		for (const n of uniqueNodes) n.label = baseLabel(n);

		// Edge ids are from|to so unchanged edges survive updates
		const seenEdges = new Set<string>();
		const processedEdges: any[] = [];
		for (const e of edges) {
			const id = edgeId(e);
			if (seenEdges.has(id)) continue;
			seenEdges.add(id);
			processedEdges.push({ ...e, id });
		}

		return { uniqueNodes, processedEdges };
	}

	// --- orbit animation ---
	// Positions are evaluated analytically from the galaxy model each frame and
	// pushed with network.moveNode(); there is no physics simulation at all, so
	// nothing can stall or run away regardless of graph size.
	let model: GalaxyModel | null = null;
	const positions = new Map<string, { x: number; y: number }>();
	let simTime = 0;
	let lastTs = 0;
	let rafId = 0;
	let running = false;
	let orbitEnabled = true;
	let reducedMotion = false;
	let hoverPaused = false;
	let dragPaused = false;
	let slowFrames = 0;
	let morph: { from: Map<string, { x: number; y: number }>; start: number; dur: number } | null =
		null;

	function ensureLoop() {
		if (running || !network || !model) return;
		running = true;
		lastTs = performance.now();
		rafId = requestAnimationFrame(frame);
	}

	function stopLoop() {
		running = false;
		cancelAnimationFrame(rafId);
	}

	function frame(ts: number) {
		if (!running || !network || !model) {
			running = false;
			return;
		}
		const dt = Math.min(0.05, Math.max(0, (ts - lastTs) / 1000));
		lastTs = ts;
		if (orbitEnabled && !hoverPaused && !dragPaused) simTime += dt;

		computePositions(model, simTime, positions);

		let k = 1;
		if (morph) {
			k = Math.min(1, (ts - morph.start) / morph.dur);
			k = 1 - Math.pow(1 - k, 3); // easeOutCubic
		}
		for (const [id, p] of positions) {
			let x = p.x;
			let y = p.y;
			if (morph && k < 1) {
				const f = morph.from.get(id);
				if (f) {
					x = f.x + (x - f.x) * k;
					y = f.y + (y - f.y) * k;
				}
			}
			network.moveNode(id, x, y);
		}
		if (morph && k >= 1) morph = null;
		network.redraw();

		// Perf watchdog: if we can't hold ~22fps for a sustained stretch, stop
		// the ambient orbit — the layout is already complete and static.
		if (dt >= 0.045) {
			if (++slowFrames > 60) orbitEnabled = false;
		} else if (slowFrames > 0) {
			slowFrames--;
		}

		if (!morph && (!orbitEnabled || hoverPaused || dragPaused)) {
			running = false;
			return;
		}
		rafId = requestAnimationFrame(frame);
	}

	function fitToModel() {
		if (!model || !container || !network) return;
		const size = 2 * (model.extent + 120);
		const scale = Math.min(container.clientWidth || 800, container.clientHeight || 600) / size;
		network.moveTo({
			position: { x: 0, y: 0 },
			scale: Math.min(1, Math.max(0.03, scale)),
			animation: false
		});
	}

	// Build the model for new data and morph from wherever nodes currently are.
	// First load morphs everything outward from the origin ("galaxy forming");
	// filter changes morph surviving nodes to their new spots and spawn added
	// nodes from their cluster's center.
	function applyGraph(nodes: GraphNode[], edges: GraphEdge[]) {
		if (!network || !nodeDataSet || !edgeDataSet) return;
		resetHighlights();

		const { uniqueNodes, processedEdges } = prepareData(nodes, edges);
		const isInitial = nodeDataSet.length === 0;
		model = computeGalaxyModel(uniqueNodes, processedEdges);
		computePositions(model, simTime, positions);

		const from = new Map<string, { x: number; y: number }>();
		if (!isInitial) {
			const existing = network.getPositions();
			for (const id of Object.keys(existing)) from.set(id, existing[id]);
		}
		for (const n of uniqueNodes) {
			const id = String(n.id);
			if (from.has(id)) continue;
			const entry = model.byId.get(id);
			from.set(id, isInitial || !entry ? { x: 0, y: 0 } : clusterCenter(entry.cluster, simTime));
		}

		// Sync datasets; new/updated nodes get their morph-start position so
		// nothing flashes at a random spot before the first animation frame.
		const nextIds = new Set(uniqueNodes.map((n) => String(n.id)));
		const removeNodes = nodeDataSet.getIds().filter((id: any) => !nextIds.has(String(id)));
		if (removeNodes.length) {
			nodeDataSet.remove(removeNodes);
			for (const id of removeNodes) tooltipImages.delete(id);
		}
		if (uniqueNodes.length) {
			nodeDataSet.update(
				uniqueNodes.map((n) => {
					const f = from.get(String(n.id))!;
					return { ...n, x: f.x, y: f.y };
				})
			);
		}

		const nextEdgeIds = new Set(processedEdges.map((e) => String(e.id)));
		const removeEdges = edgeDataSet.getIds().filter((id: any) => !nextEdgeIds.has(String(id)));
		if (removeEdges.length) edgeDataSet.remove(removeEdges);
		if (processedEdges.length) edgeDataSet.update(processedEdges);

		morph = reducedMotion
			? null
			: { from, start: performance.now(), dur: isInitial ? 1600 : 700 };

		if (uniqueNodes.length) loading = false;
		if (isInitial && uniqueNodes.length) fitToModel();
		if (uniqueNodes.length) ensureLoop();
		else network.redraw();
	}

	// After a drag, fold the node's new position back into the orbital model so
	// the animation continues from where the user dropped it instead of
	// snapping back.
	function commitDragToModel() {
		const anchorId = dragState.anchorId;
		if (anchorId == null || !model || !network) return;
		const entry = model.byId.get(String(anchorId));
		if (!entry) return;
		const dropped = network.getPositions([anchorId])[anchorId];
		const computed = positions.get(String(anchorId));
		if (!dropped || !computed) return;
		const { cluster, node } = entry;
		if (node.parent === null) {
			cluster.ox += dropped.x - computed.x;
			cluster.oy += dropped.y - computed.y;
		} else {
			const parentPos = positions.get(node.parent);
			if (!parentPos) return;
			const dx = dropped.x - parentPos.x;
			const dy = dropped.y - parentPos.y;
			node.r = Math.hypot(dx, dy);
			node.a = Math.atan2(dy, dx) - node.w * simTime;
		}
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

	// Inject each node's live position into a DataSet update so the update
	// itself doesn't teleport the node to a stale stored x/y.
	function withCurrentPositions(items: any[]) {
		if (!items.length || !network) return items;
		const pos = network.getPositions(items.map((n: any) => n.id));
		return items.map((n: any) => {
			const p = pos[n.id];
			return p ? { ...n, x: p.x, y: p.y } : n;
		});
	}

	// --- highlight helpers (whole subtree) ---
	function resetHighlights() {
		if (!nodeDataSet || !edgeDataSet) return;

		if (lastHighlightedNodes.length) {
			const resetNodes = lastHighlightedNodes
				.map((id) => nodeDataSet.get(id))
				.filter(Boolean)
				.map(resetNodeToBase);
			if (resetNodes.length) nodeDataSet.update(withCurrentPositions(resetNodes));
			lastHighlightedNodes = [];
		}

		if (lastHighlightedEdges.length) {
			const resetEdges = lastHighlightedEdges
				.map((id) => edgeDataSet.get(id))
				.filter(Boolean)
				.map((e: any) => ({ ...e, color: BASE_EDGE_COLOR, width: 1.5 }));
			if (resetEdges.length) edgeDataSet.update(resetEdges);
			lastHighlightedEdges = [];
		}
	}

	function getDescendantsAndEdges(rootId: string | number) {
		// BFS over directed edges: follow e.from -> e.to
		const allEdges: any[] = edgeDataSet.get();
		const outMap = new Map<string, Array<any>>();
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

		for (let qi = 0; qi < q.length; qi++) {
			const cur = q[qi];
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

		const updatedEdges = edges.map((e) => ({
			...e,
			color: { color: blue, highlight: blue, hover: blue },
			width: 2.5
		}));
		if (updatedEdges.length) edgeDataSet.update(updatedEdges);

		// Small subtrees get their labels back while highlighted, even when
		// leaf labels are globally hidden for performance
		const showLabels = labelsVisible || nodeIds.length <= HIGHLIGHT_LABEL_LIMIT;
		const updatedNodes = nodeIds
			.map((id) => nodeDataSet.get(id))
			.filter(Boolean)
			.map((n: any) => ({
				...n,
				label: showLabels ? (n.fullLabel ?? n.label) : n.label,
				color: {
					...(n.color ?? {}),
					border: blue,
					background: 'rgba(59,130,246,0.15)',
					highlight: { border: blue, background: 'rgba(59,130,246,0.22)' }
				}
			}));
		if (updatedNodes.length) nodeDataSet.update(withCurrentPositions(updatedNodes));

		lastHighlightedNodes = nodeIds;
		lastHighlightedEdges = edgeIds;
	}

	let loadingFailsafe: ReturnType<typeof setTimeout> | undefined;

	onMount(async () => {
		const vis: any = await import('vis-network/standalone');

		reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches ?? false;
		if (reducedMotion) orbitEnabled = false;

		nodeDataSet = new vis.DataSet([]);
		edgeDataSet = new vis.DataSet([]);

		const options = {
			// Layout and animation are fully deterministic (galaxyLayout.ts), so
			// vis physics stays off: no stabilization phase, no runaway simulation.
			layout: { improvedLayout: false },
			physics: false,
			nodes: {
				shape: 'dot',
				size: 14,
				font: DEFAULT_NODE_FONT,
				borderWidth: 2,
				color: DEFAULT_NODE_COLOR
			},
			groups: {
				domain: {
					color: GROUP_COLORS.domain,
					shape: 'diamond',
					size: 36,
					font: { size: 16, color: '#fff' }
				},
				subdomain: {
					color: GROUP_COLORS.subdomain,
					shape: 'dot',
					size: 18,
					font: { size: 14, color: '#fff' }
				},
				site: {
					color: GROUP_COLORS.site,
					shape: 'dot',
					size: 16,
					font: { size: 14, color: '#fff' }
				},
				error: {
					color: GROUP_COLORS.error,
					shape: 'triangle',
					size: 16,
					font: { size: 14, color: '#fff' }
				}
			},
			edges: {
				width: 1.5,
				smooth: false,
				shadow: { enabled: false },
				color: BASE_EDGE_COLOR,
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
			}
		} as any;

		network = new vis.Network(container, { nodes: nodeDataSet, edges: edgeDataSet }, options);

		// --- Interactions ---
		// Click: background clears; node click highlights whole subtree and opens site URLs
		network.on('click', (params: any) => {
			if (!params.nodes?.length) {
				resetHighlights();
				network.unselectAll();
				return;
			}

			const nodeId = params.nodes[0];
			const n = nodeDataSet.get(nodeId);

			highlightSubtree(nodeId);

			// Click-to-open for site nodes
			if (n?.group === 'site') {
				const url = String(n.meta?.url ?? n.fullLabel ?? '');
				if (url) window.open(url.startsWith('http') ? url : `https://${url}`, '_blank');
			}
		});

		network.on('deselectNode', () => {
			resetHighlights();
		});

		// Hover: pause the orbit so the node doesn't drift out from under the
		// tooltip, and lazily load the screenshot on first hover
		network.on('hoverNode', (params: any) => {
			hoverPaused = true;
			const img = tooltipImages.get(params.node);
			if (img?.dataset.src && !img.getAttribute('src')) {
				img.src = img.dataset.src;
			}
		});
		network.on('blurNode', () => {
			hoverPaused = false;
			ensureLoop();
		});

		// Drag a node and its whole subtree as a unit; orbit pauses meanwhile
		network.on('dragStart', (params: any) => {
			if (!params.nodes?.length) return;
			dragPaused = true;
			const anchorId = params.nodes[0];
			const descendants = getDescendantsAndEdges(anchorId).nodeIds;
			const pos = network.getPositions([anchorId, ...descendants]);
			// Delta is measured pointer-to-pointer, not pointer-to-node-center,
			// so children don't jump by the grab offset
			dragState = {
				anchorId,
				pointerStart: params.pointer.canvas,
				startPos: pos,
				childIds: descendants
			};
		});

		network.on('dragging', (params: any) => {
			if (dragState.anchorId == null || !dragState.pointerStart) return;
			const cur = params.pointer.canvas;
			const dx = cur.x - dragState.pointerStart.x;
			const dy = cur.y - dragState.pointerStart.y;
			for (const cid of dragState.childIds) {
				const cStart = dragState.startPos[String(cid)];
				if (cStart) network.moveNode(cid, cStart.x + dx, cStart.y + dy);
			}
		});

		network.on('dragEnd', () => {
			if (dragState.anchorId != null) commitDragToModel();
			dragState = { anchorId: null, pointerStart: null, startPos: {}, childIds: [] };
			dragPaused = false;
			ensureLoop();
		});

		// React to data (initial load, filter changes): fires immediately with
		// the store's current value, then on every update
		unsubscribe = graphData.subscribe(({ nodes, edges }) => {
			applyGraph(nodes || [], edges || []);
		});

		// Never spin forever if the dataset stays empty
		loadingFailsafe = setTimeout(() => (loading = false), 8000);
	});

	onDestroy(() => {
		stopLoop();
		clearTimeout(loadingFailsafe);
		try {
			unsubscribe?.();
		} catch {}
		try {
			network?.off('click');
			network?.off('dragStart');
			network?.off('dragging');
			network?.off('dragEnd');
			network?.off('deselectNode');
			network?.off('hoverNode');
			network?.off('blurNode');
		} catch {}
		try {
			network?.destroy();
		} catch {}
		try {
			nodeDataSet?.clear();
			edgeDataSet?.clear();
		} catch {}
		tooltipImages.clear();
		positions.clear();
		model = null;
	});
</script>

<div class="graph-wrapper">
	<div bind:this={container} class="w-full h-[80vh] rounded bg-transparent"></div>

	{#if loading}
		<div class="graph-loading" role="status" aria-live="polite" aria-label="Loading graph">
			<div class="spinner" aria-hidden="true"></div>
			<div class="label">Loading graph…</div>
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
