<script lang="ts">
	import { run } from 'svelte/legacy';

	import { onMount, onDestroy, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { MapPin, Network, Globe } from 'lucide-svelte';
	import { statusFilter } from '$lib/stores/filterStore';
	import 'leaflet/dist/leaflet.css';

	type GeoInfo = {
		ip: string;
		country: string;
		country_code: string;
		city: string;
		region: string;
		lat: number;
		lon: number;
		org: string;
		isp: string;
		as: string;
	};

	type ServerGeo = {
		server_id: number;
		url: string;
		ips: string[];
		status: string;
	};

	type ServerGeoResponse = {
		servers: ServerGeo[];
		geo: Record<string, GeoInfo>;
		local_ips: string[];
	};

	let servers: ServerGeo[] = $state([]);
	let geoMap: Record<string, GeoInfo> = $state({});
	let localIPs: string[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let mapContainer: HTMLDivElement = $state();
	let map: any = $state(null);
	let view: 'map' | 'cluster' = $state('map');
	let selectedCountry: string | null = $state(null);

	let L: any = $state(null);
	let markersLayer: any = $state(null);

	onMount(async () => {
		try {
			const [resp, leaflet] = await Promise.all([fetch('/api/servers/geo'), import('leaflet')]);
			if (!resp.ok) throw new Error('Failed to fetch geo data');
			const data: ServerGeoResponse = await resp.json();
			servers = data.servers;
			geoMap = data.geo;
			localIPs = data.local_ips || [];
			L = leaflet.default || leaflet;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load';
		} finally {
			loading = false;
		}
	});

	onDestroy(() => {
		if (map) {
			map.remove();
			map = null;
			markersLayer = null;
		}
	});

	function getFlagEmoji(cc: string): string {
		if (!cc || cc.length !== 2) return '';
		return String.fromCodePoint(
			...[...cc.toUpperCase()].map((c) => 0x1f1e6 + c.charCodeAt(0) - 65)
		);
	}

	function buildMap(L: any) {
		if (map) {
			map.remove();
			map = null;
		}
		if (!mapContainer) return;

		map = L.map(mapContainer).setView([62, 10], 5);

		L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
			attribution:
				'&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a> &copy; <a href="https://carto.com/">CARTO</a>',
			subdomains: 'abcd',
			maxZoom: 19
		}).addTo(map);

		for (const loc of Object.values(locationGroups)) {
			const totalServers = loc.ips.reduce((s, g) => s + g.servers.length, 0);
			const allUp = loc.ips.every((g) => g.servers.every((s) => s.status === 'up'));
			const anyDown = loc.ips.some((g) => g.servers.some((s) => s.status === 'down'));

			const color = anyDown ? '#ef4444' : allUp ? '#22c55e' : '#eab308';
			const radius = Math.max(8, Math.min(20, 6 + totalServers * 2));

			const circle = L.circleMarker([loc.lat, loc.lon], {
				radius,
				fillColor: color,
				color: color,
				weight: 1,
				opacity: 0.8,
				fillOpacity: 0.4
			}).addTo(map);

			const ipList = loc.ips
				.map((g) => {
					const domains = g.servers
						.map(
							(s) =>
								`<a href="/server/${s.server_id}" style="color:#93c5fd;text-decoration:none">${s.url}</a> <span style="color:${s.status === 'up' ? '#4ade80' : '#f87171'}">${s.status}</span>`
						)
						.join('<br/>');
					return `<div style="margin-bottom:0.5rem">
					<div style="font-family:monospace;color:#e5e7eb;font-size:0.8125rem">${g.ip}</div>
					<div style="font-size:0.6875rem;color:#9ca3af">${g.geo?.org || ''} ${g.geo?.as || ''}</div>
					<div style="margin-top:0.25rem">${domains}</div>
				</div>`;
				})
				.join('');

			circle.bindPopup(
				`
				<div style="background:#1a1a1a;color:#d1d5db;padding:0.5rem;border-radius:0.375rem;min-width:200px">
					<div style="font-weight:600;margin-bottom:0.5rem;color:#e5e7eb">${loc.city}, ${loc.country}</div>
					<div style="font-size:0.75rem;color:#6b7280;margin-bottom:0.5rem">${totalServers} server${totalServers > 1 ? 's' : ''} on ${loc.ips.length} IP${loc.ips.length > 1 ? 's' : ''}</div>
					${ipList}
				</div>
			`,
				{ className: 'dark-popup', maxWidth: 350 }
			);
		}

		// Fit bounds if we have markers
		const coords = Object.values(locationGroups).map((l) => [l.lat, l.lon] as [number, number]);
		if (coords.length > 1) {
			map.fitBounds(L.latLngBounds(coords).pad(0.2));
		} else if (coords.length === 1) {
			map.setView(coords[0], 8);
		}

		// Add "You are here" marker via browser geolocation
		if (navigator.geolocation) {
			navigator.geolocation.getCurrentPosition(
				(pos) => {
					if (!map) return;
					const { latitude, longitude } = pos.coords;

					const youIcon = L.divIcon({
						className: 'you-marker',
						html: `<div style="
						width: 14px; height: 14px;
						background: #3b82f6;
						border: 2px solid #fff;
						border-radius: 50%;
						box-shadow: 0 0 8px rgba(59,130,246,0.6);
					"></div>`,
						iconSize: [14, 14],
						iconAnchor: [7, 7]
					});

					L.marker([latitude, longitude], { icon: youIcon })
						.addTo(map)
						.bindPopup(
							'<div style="color:#d1d5db;background:#1a1a1a;padding:0.25rem 0.5rem;border-radius:0.25rem;font-size:0.8125rem">You are here</div>',
							{ className: 'dark-popup' }
						);
				},
				() => {
					// Geolocation denied or unavailable — silently ignore
				}
			);
		}
	}
	let filteredServers = $derived(
		(() => {
			if ($statusFilter === 'all') return servers;
			if ($statusFilter === 'online') return servers.filter((s) => s.status === 'up');
			if ($statusFilter === 'issues') return servers.filter((s) => s.status === 'down');
			if ($statusFilter === 'new') return servers.filter((s) => s.status === 'unknown');
			return servers;
		})()
	);
	// Flatten: group by IP across all servers
	let ipGroups = $derived(
		filteredServers.reduce(
			(acc, s) => {
				for (const ip of s.ips || []) {
					if (!acc[ip]) acc[ip] = { ip, geo: geoMap[ip], servers: [] };
					acc[ip].servers.push(s);
				}
				return acc;
			},
			{} as Record<string, { ip: string; geo?: GeoInfo; servers: ServerGeo[] }>
		)
	);
	let allLocationGroups = $derived(
		Object.values(ipGroups).reduce(
			(acc, group) => {
				if (!group.geo) return acc;
				const key = `${group.geo.lat},${group.geo.lon}`;
				if (!acc[key])
					acc[key] = {
						lat: group.geo.lat,
						lon: group.geo.lon,
						city: group.geo.city,
						country: group.geo.country,
						country_code: group.geo.country_code,
						ips: []
					};
				acc[key].ips.push(group);
				return acc;
			},
			{} as Record<
				string,
				{
					lat: number;
					lon: number;
					city: string;
					country: string;
					country_code: string;
					ips: (typeof ipGroups)[string][];
				}
			>
		)
	);

	let locationGroups = $derived(
		selectedCountry
			? Object.fromEntries(
					Object.entries(allLocationGroups).filter(
						([_, loc]) => loc.country_code === selectedCountry
					)
				)
			: allLocationGroups
	);
	// Reset map when switching to clusters
	run(() => {
		if (view === 'cluster' && map) {
			map.remove();
			map = null;
			markersLayer = null;
		}
	});
	// Initialize the map once when ready
	run(() => {
		if (!loading && L && view === 'map' && mapContainer && !map) {
			tick().then(() => buildMap(L));
		}
	});
	// Rebuild markers when filtered servers or selected country changes
	run(() => {
		if (map && L && (filteredServers || selectedCountry !== undefined)) {
			buildMap(L);
		}
	});

	function handleCountryClick(countryCode: string) {
		if (selectedCountry === countryCode) {
			selectedCountry = null;
		} else {
			selectedCountry = countryCode;
			// If selecting LOCAL, switch to cluster view since local IPs can't be shown on map
			if (countryCode === 'LOCAL') {
				view = 'cluster';
			}
		}
	}
</script>

<div class="page">
	<div class="header">
		<h1 class="title">
			<Globe size={24} />
			Infrastructure Map
		</h1>
		<div class="header-controls">
			<div class="view-toggle">
				<button class:active={$statusFilter === 'all'} onclick={() => ($statusFilter = 'all')}>
					All <span class="filter-count">{servers.length}</span>
				</button>
				<button
					class:active={$statusFilter === 'online'}
					onclick={() => ($statusFilter = 'online')}
				>
					Online <span class="filter-count">{servers.filter((s) => s.status === 'up').length}</span>
				</button>
				<button
					class:active={$statusFilter === 'issues'}
					onclick={() => ($statusFilter = 'issues')}
				>
					Offline <span class="filter-count"
						>{servers.filter((s) => s.status === 'down').length}</span
					>
				</button>
				<button class:active={$statusFilter === 'new'} onclick={() => ($statusFilter = 'new')}>
					New <span class="filter-count"
						>{servers.filter((s) => s.status === 'unknown').length}</span
					>
				</button>
			</div>
			<div class="view-toggle">
				<button class:active={view === 'map'} onclick={() => (view = 'map')}>
					<MapPin size={14} /> Map
				</button>
				<button class:active={view === 'cluster'} onclick={() => (view = 'cluster')}>
					<Network size={14} /> IP Clusters
				</button>
			</div>
		</div>
	</div>

	{#if loading}
		<div class="loading">Loading geo data...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else}
		{#if view === 'map'}
			<div class="map-wrap" bind:this={mapContainer}></div>
		{:else}
			<div class="clusters">
				{#each Object.values(ipGroups)
					.sort((a, b) => b.servers.length - a.servers.length)
					.filter( (group) => (selectedCountry === 'LOCAL' ? localIPs.includes(group.ip) : selectedCountry ? group.geo?.country_code === selectedCountry : true) ) as group}
					{@const isLocal = localIPs.includes(group.ip)}
					<div class="cluster-card" class:local-cluster={isLocal}>
						<div class="cluster-header">
							<div style="display: flex; align-items: center; gap: 0.5rem;">
								{#if isLocal}
									<span class="local-pill">Local</span>
								{/if}
								<span class="cluster-ip">{group.ip}</span>
							</div>
							<span class="cluster-count"
								>{group.servers.length} domain{group.servers.length > 1 ? 's' : ''}</span
							>
						</div>
						{#if group.geo}
							<div class="cluster-geo">
								{group.geo.city}, {group.geo.country} &middot; {group.geo.org} &middot; {group.geo
									.as}
							</div>
						{:else if isLocal}
							<div class="cluster-geo">Private/Local Network</div>
						{/if}
						<div class="cluster-domains">
							{#each group.servers as s}
								<button class="domain-row" onclick={() => goto(`/server/${s.server_id}`)}>
									<span
										class="domain-dot"
										class:up={s.status === 'up'}
										class:down={s.status === 'down'}
									></span>
									<span class="domain-url">{s.url}</span>
									<span class="domain-status">{s.status}</span>
								</button>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		{/if}

		{#if Object.keys(allLocationGroups).length > 0 || localIPs.length > 0}
			<div class="country-strip">
				{#each Object.entries(Object.values(allLocationGroups).reduce((acc, loc) => {
						const cc = loc.country_code || '??';
						if (!acc[cc]) acc[cc] = { country: loc.country, count: 0, cities: new Set() };
						acc[cc].count += loc.ips.reduce((s, g) => s + g.servers.length, 0);
						acc[cc].cities.add(loc.city);
						return acc;
					}, {})).sort((a, b) => b[1].count - a[1].count) as [cc, info]}
					<button
						class="country-chip"
						class:selected={selectedCountry === cc}
						onclick={() => handleCountryClick(cc)}
					>
						<span class="country-flag">{getFlagEmoji(cc)}</span>
						<span class="country-name">{info.country}</span>
						<span class="country-count">{info.count}</span>
					</button>
				{/each}
				{#if localIPs.length > 0}
					<button
						class="country-chip"
						class:selected={selectedCountry === 'LOCAL'}
						onclick={() => handleCountryClick('LOCAL')}
					>
						<span class="country-name">Local</span>
						<span class="country-count">{localIPs.length}</span>
					</button>
				{/if}
			</div>
		{/if}
	{/if}
</div>

<style>
	:global(.dark-popup .leaflet-popup-content-wrapper) {
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 0.5rem;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
	}
	:global(.dark-popup .leaflet-popup-tip) {
		background: #1a1a1a;
		border: 1px solid #333;
	}
	:global(.dark-popup .leaflet-popup-close-button) {
		color: #9ca3af;
	}

	.page {
		padding: 1rem;
		/* width: 97%; */
		height: 100%;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		overflow: hidden;
		box-sizing: border-box;
	}

	.header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: #202020;
		border-radius: 0.5rem;
		padding: 1rem;
	}

	.title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 1.25rem;
		font-weight: 500;
		color: #e5e7eb;
	}

	.title :global(svg) {
		color: #4ade80;
	}

	.header-controls {
		display: flex;
		gap: 0.5rem;
	}

	.filter-count {
		font-size: 0.6875rem;
		color: #6b7280;
		font-variant-numeric: tabular-nums;
	}

	.view-toggle {
		display: flex;
		gap: 0.25rem;
		background: #2b2b2b;
		border-radius: 0.375rem;
		padding: 0.25rem;
	}

	.view-toggle button {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.75rem;
		border: none;
		background: none;
		color: #9ca3af;
		font-size: 0.8125rem;
		border-radius: 0.25rem;
		cursor: pointer;
		transition: all 0.15s;
	}

	.view-toggle button:hover {
		color: #e5e7eb;
	}
	.view-toggle button.active {
		background: #404040;
		color: #e5e7eb;
	}

	.loading,
	.error {
		text-align: center;
		padding: 4rem;
		color: #9ca3af;
		font-size: 0.875rem;
	}

	.error {
		color: #ef4444;
	}

	.country-strip {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		padding: 0.75rem 0;
	}

	.country-chip {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.625rem;
		background: #202020;
		border: 1px solid transparent;
		border-radius: 0.375rem;
		font-size: 0.8125rem;
		cursor: pointer;
		transition: all 0.15s;
	}

	.country-chip:hover {
		background: #2b2b2b;
		border-color: #404040;
	}

	.country-chip.selected {
		background: #2563eb;
		border-color: #3b82f6;
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
	}

	.country-chip.selected .country-name {
		color: #ffffff;
	}

	.country-chip.selected .country-count {
		color: #dbeafe;
	}

	.country-flag {
		font-size: 1rem;
	}

	.country-name {
		color: #d1d5db;
	}

	.country-count {
		color: #6b7280;
		font-size: 0.75rem;
		font-variant-numeric: tabular-nums;
	}

	.map-wrap {
		flex: 1;
		min-height: 0;
		border-radius: 0.5rem;
		overflow: hidden;
		background: #1a1a1a;
	}

	.clusters {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: 0.75rem;
		flex: 1;
		min-height: 0;
		overflow-y: auto;
	}

	.cluster-card {
		background: #202020;
		border-radius: 0.5rem;
		padding: 1rem;
	}

	.cluster-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.375rem;
	}

	.cluster-ip {
		font-family: ui-monospace, monospace;
		font-size: 0.875rem;
		color: #e5e7eb;
		font-weight: 500;
	}

	.cluster-count {
		font-size: 0.6875rem;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.cluster-geo {
		font-size: 0.75rem;
		color: #6b7280;
		margin-bottom: 0.75rem;
	}

	.cluster-domains {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.domain-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.375rem 0.5rem;
		background: none;
		border: none;
		border-radius: 0.25rem;
		cursor: pointer;
		text-align: left;
		width: 100%;
		transition: background 0.15s;
	}

	.domain-row:hover {
		background: #2b2b2b;
	}

	.domain-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
		background: #6b7280;
	}

	.domain-dot.up {
		background: #22c55e;
	}
	.domain-dot.down {
		background: #ef4444;
	}

	.domain-url {
		font-size: 0.8125rem;
		color: #d1d5db;
		flex: 1;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.domain-status {
		font-size: 0.6875rem;
		color: #6b7280;
		text-transform: uppercase;
	}

	.local-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
		border-radius: 0.25rem;
		font-size: 0.6875rem;
		font-weight: 500;
		color: #ffffff;
	}
</style>
