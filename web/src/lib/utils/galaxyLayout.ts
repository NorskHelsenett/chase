// Deterministic "galaxy" layout for the site graph.
//
// The biggest cluster (root domain with the most nodes) sits at the origin —
// the galaxy. Every other cluster is a planet packed onto concentric orbit
// rings around it. Inside a cluster, each domain hub holds its leaf sites on
// a sunflower disc, and sub-domain hubs sit further out like moons with
// discs of their own.
//
// The model is fully analytic: a node's position at time t derives from polar
// coordinates relative to its parent, so animating 3000 nodes is O(n) trig
// per frame — no physics simulation, nothing to stabilize, nothing to stall.

export const GOLDEN_ANGLE = 2.399963229728653;

export interface LayoutNode {
	id: string | number;
	cluster?: string;
}
export interface LayoutEdge {
	from: string | number;
	to: string | number;
}

export interface OrbitNode {
	id: string;
	/** id of the node this one orbits; null = the cluster center */
	parent: string | null;
	/** polar coordinates relative to the parent */
	r: number;
	a: number;
	/** angular velocity in rad/s */
	w: number;
}

export interface OrbitCluster {
	key: string;
	/** bounding radius of the cluster's internal layout */
	radius: number;
	/** orbit around the galaxy origin */
	orbitR: number;
	orbitA: number;
	orbitW: number;
	/** persistent offset from user drags */
	ox: number;
	oy: number;
	/** parent-before-child order, so positions resolve in one pass */
	nodes: OrbitNode[];
}

export interface GalaxyModel {
	clusters: OrbitCluster[];
	/** max distance from origin, for fitting the viewport */
	extent: number;
	byId: Map<string, { cluster: OrbitCluster; node: OrbitNode }>;
}

const LEAF_BASE = 46; // first sunflower ring distance from its hub
const LEAF_STEP = 26; // sunflower radial step (controls leaf density)
const HUB_GAP = 80; // gap between a hub's leaf disc and its child hubs
const CHILD_PAD = 40; // spacing between sibling hub subtrees
const SINGLE_RADIUS = 26; // bounding radius of a childless node
const RING_GAP = 160; // gap between planet orbit rings
const PLANET_PAD = 60; // spacing between planets on the same ring
const LEAF_SPIN = (2 * Math.PI) / 150; // leaf disc revolution: 2.5 min
const HUB_SPIN = (2 * Math.PI) / 420; // hub orbit around its parent: 7 min
const RING_PERIOD = 260; // innermost planet ring revolution, seconds

function hashAngle(key: string): number {
	let h = 0;
	for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) | 0;
	return ((h >>> 0) % 360) * (Math.PI / 180);
}

export function clusterCenter(c: OrbitCluster, t: number): { x: number; y: number } {
	const a = c.orbitA + c.orbitW * t;
	return { x: Math.cos(a) * c.orbitR + c.ox, y: Math.sin(a) * c.orbitR + c.oy };
}

/** Evaluate every node's position at time t into `out` (reused between frames). */
export function computePositions(
	model: GalaxyModel,
	t: number,
	out: Map<string, { x: number; y: number }>
): void {
	out.clear();
	for (const c of model.clusters) {
		const center = clusterCenter(c, t);
		for (const n of c.nodes) {
			const p = (n.parent !== null && out.get(n.parent)) || center;
			const a = n.a + n.w * t;
			out.set(n.id, { x: p.x + Math.cos(a) * n.r, y: p.y + Math.sin(a) * n.r });
		}
	}
}

export function computeGalaxyModel(nodes: LayoutNode[], edges: LayoutEdge[]): GalaxyModel {
	const ids = new Set(nodes.map((n) => String(n.id)));

	// Parent/children maps. First parent wins; self-loops, dangling edges and
	// cycles are ignored so the layout always works on a forest.
	const parentOf = new Map<string, string>();
	const childrenOf = new Map<string, string[]>();
	for (const e of edges) {
		const from = String(e.from);
		const to = String(e.to);
		if (from === to || !ids.has(from) || !ids.has(to) || parentOf.has(to)) continue;
		let ancestor: string | undefined = from;
		let cyclic = false;
		while (ancestor !== undefined) {
			if (ancestor === to) {
				cyclic = true;
				break;
			}
			ancestor = parentOf.get(ancestor);
		}
		if (cyclic) continue;
		parentOf.set(to, from);
		const arr = childrenOf.get(from) ?? [];
		arr.push(to);
		childrenOf.set(from, arr);
	}

	// Group nodes by cluster (root domain)
	const clusterMembers = new Map<string, string[]>();
	for (const n of nodes) {
		const key = String(n.cluster ?? n.id);
		const arr = clusterMembers.get(key) ?? [];
		arr.push(String(n.id));
		clusterMembers.set(key, arr);
	}

	function layoutCluster(key: string, members: string[]): OrbitCluster {
		const memberSet = new Set(members);
		const placements = new Map<string, OrbitNode>();

		// Places `kids` around `parent` (null = cluster center): leaves on a
		// sunflower disc, hub children on a ring outside it, each with an
		// angular sector proportional to its subtree size. Returns the
		// bounding radius of everything placed.
		function layoutChildren(parent: string | null, kids: string[], sign: number): number {
			if (!kids.length) return SINGLE_RADIUS;
			const inCluster = kids.filter((k) => memberSet.has(k));
			const leaves = inCluster.filter((k) => !childrenOf.get(k)?.length);
			const hubs = inCluster.filter((k) => childrenOf.get(k)?.length);
			const phase = hashAngle(parent ?? key);

			leaves.forEach((k, j) => {
				placements.set(k, {
					id: k,
					parent,
					r: LEAF_BASE + LEAF_STEP * Math.sqrt(j),
					a: phase + j * GOLDEN_ANGLE,
					w: sign * LEAF_SPIN
				});
			});
			let radius = leaves.length
				? LEAF_BASE + LEAF_STEP * Math.sqrt(leaves.length)
				: SINGLE_RADIUS;
			if (!hubs.length) return radius;

			const subRadii = hubs.map((k) => {
				const r = layoutChildren(k, childrenOf.get(k) ?? [], -sign);
				return Math.max(r, SINGLE_RADIUS);
			});
			// Far enough out that all subtrees fit side by side angularly
			let dist = radius + HUB_GAP + Math.max(...subRadii);
			const needed = subRadii.reduce(
				(s, r) => s + 2 * Math.asin(Math.min(0.95, (r + CHILD_PAD) / dist)),
				0
			);
			if (needed > 2 * Math.PI) dist *= needed / (2 * Math.PI);

			const total = subRadii.reduce((s, r) => s + r + CHILD_PAD, 0);
			let acc = phase;
			hubs.forEach((k, i) => {
				const width = (2 * Math.PI * (subRadii[i] + CHILD_PAD)) / total;
				placements.set(k, { id: k, parent, r: dist, a: acc + width / 2, w: sign * HUB_SPIN });
				acc += width;
				radius = Math.max(radius, dist + subRadii[i]);
			});
			return radius;
		}

		// Roots: no parent, or parent outside this cluster
		const roots = members.filter((id) => {
			const p = parentOf.get(id);
			return !p || !memberSet.has(p);
		});

		let radius: number;
		if (roots.length === 1) {
			const root = roots[0];
			placements.set(root, { id: root, parent: null, r: 0, a: 0, w: 0 });
			radius = layoutChildren(root, childrenOf.get(root) ?? [], 1);
		} else {
			radius = layoutChildren(null, roots, 1);
		}

		// Safety net: anything unreachable (shouldn't happen on forest data)
		// gets parked on the cluster's edge instead of being dropped.
		const missing = members.filter((id) => !placements.has(id));
		missing.forEach((k, j) => {
			placements.set(k, {
				id: k,
				parent: null,
				r: radius + HUB_GAP + LEAF_STEP * Math.sqrt(j),
				a: j * GOLDEN_ANGLE,
				w: 0
			});
		});
		if (missing.length) radius += HUB_GAP + LEAF_STEP * Math.sqrt(missing.length);

		// Flatten parent-before-child so per-frame evaluation is a single pass
		const byParent = new Map<string | null, OrbitNode[]>();
		for (const p of placements.values()) {
			const arr = byParent.get(p.parent) ?? [];
			arr.push(p);
			byParent.set(p.parent, arr);
		}
		const ordered: OrbitNode[] = [...(byParent.get(null) ?? [])];
		for (let i = 0; i < ordered.length; i++) {
			ordered.push(...(byParent.get(ordered[i].id) ?? []));
		}

		return { key, radius, orbitR: 0, orbitA: 0, orbitW: 0, ox: 0, oy: 0, nodes: ordered };
	}

	const clusters = [...clusterMembers.entries()].map(([key, members]) =>
		layoutCluster(key, members)
	);

	// Biggest cluster is the galaxy at the origin; the rest are planets packed
	// onto concentric rings around it, largest planets on the inner rings.
	clusters.sort((a, b) => b.nodes.length - a.nodes.length);
	const planets = clusters.slice(1);
	planets.sort((a, b) => b.radius - a.radius);

	let extent = clusters.length ? clusters[0].radius : 0;
	let prevOuter = extent;
	let firstRingR = 0;
	let idx = 0;
	while (idx < planets.length) {
		const ringR = prevOuter + RING_GAP + planets[idx].radius;
		if (!firstRingR) firstRingR = ringR;

		// Greedily fill the ring while there's angular room
		const ring: OrbitCluster[] = [];
		let used = 0;
		let maxR = 0;
		while (idx < planets.length) {
			const p = planets[idx];
			const halfW = Math.asin(Math.min(0.95, (p.radius + PLANET_PAD) / ringR));
			if (ring.length && used + 2 * halfW > 2 * Math.PI) break;
			ring.push(p);
			used += 2 * halfW;
			maxR = Math.max(maxR, p.radius);
			idx++;
		}

		// Kepler-ish: outer rings orbit slower
		const w = ((2 * Math.PI) / RING_PERIOD) * Math.pow(firstRingR / ringR, 1.5);
		const slack = Math.max(0, 2 * Math.PI - used) / ring.length;
		let acc = hashAngle(`ring:${ring[0].key}`);
		for (const p of ring) {
			const halfW = Math.asin(Math.min(0.95, (p.radius + PLANET_PAD) / ringR));
			acc += halfW + slack / 2;
			p.orbitR = ringR;
			p.orbitA = acc;
			p.orbitW = w;
			acc += halfW + slack / 2;
		}
		prevOuter = ringR + maxR;
		extent = Math.max(extent, prevOuter);
	}

	const byId = new Map<string, { cluster: OrbitCluster; node: OrbitNode }>();
	for (const cluster of clusters) {
		for (const node of cluster.nodes) byId.set(node.id, { cluster, node });
	}

	return { clusters, extent: Math.max(extent, 200), byId };
}
