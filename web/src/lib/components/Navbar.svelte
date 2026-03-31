<script>
	import { onMount } from 'svelte';
	import { Home, LogIn, Logs, Grid, Share2, Globe, X } from 'lucide-svelte';
	import Avatar from './Avatar.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { isLoggedIn } from '$lib/auth';
	import { unreadCount, loadNotifications, resetNotifications } from '$lib/push/notificationStore';

	const routes = [
		{ path: '/', icon: Home, label: 'Home', auth: false },
		{ path: '/dashboard?active=true', icon: Logs, label: 'Dashboard', auth: true },
		{ path: '/grid?active=true', icon: Grid, label: 'Grid', auth: true },
		{ path: '/graph', icon: Share2, label: 'Graph', auth: true },
		{ path: '/map', icon: Globe, label: 'Map', auth: true }
	];

	let showModal = false;

	function handleLogin() {
		showModal = true;
	}

	async function toProfile() {
		await goto('/profile');
	}

	onMount(() => {
		const unsubscribe = isLoggedIn.subscribe((loggedIn) => {
			if (loggedIn) {
				loadNotifications().catch((error) => {
					console.error('Failed to load notifications:', error);
				});
			} else {
				resetNotifications();
			}
		});
		return () => unsubscribe();
	});

	// Track current path reactively
	$: currentPath = $page.url.pathname;
	$: currentFull = $page.url.pathname + $page.url.search;

	function isActive(routePath, _currentPath, _currentFull) {
		if (routePath === '/') return _currentPath === '/';
		return _currentFull === routePath || _currentPath === routePath.split('?')[0];
	}

	function formatUnread(count) {
		if (!count) return '';
		return count > 99 ? '99+' : `${count}`;
	}
</script>

<div class="navbar">
	<div class="nav-brand">
		<span class="brand-text">CHASE</span>
	</div>

	<nav class="nav-links">
		{#each routes as route}
			{#if !route.auth || $isLoggedIn}
				<button
					class="nav-link"
					class:active={isActive(route.path, currentPath, currentFull)}
					on:click={() => goto(route.path)}
				>
					<svelte:component this={route.icon} size={18} />
					<span>{route.label}</span>
				</button>
			{/if}
		{/each}
	</nav>

	<div class="nav-footer">
		{#if $isLoggedIn}
			<button class="nav-link profile-link" on:click={toProfile}>
				<Avatar />
				<span>Profile</span>
				{#if $unreadCount > 0}
					<span class="badge">{formatUnread($unreadCount)}</span>
				{/if}
			</button>
		{:else}
			<button class="nav-link" on:click={handleLogin}>
				<LogIn size={18} />
				<span>Log in</span>
			</button>
		{/if}
	</div>
</div>

{#if showModal}
	<!-- svelte-ignore a11y-click-events-have-key-events -->
	<!-- svelte-ignore a11y-no-static-element-interactions -->
	<div class="modal-backdrop" on:click={() => (showModal = false)}>
		<div class="modal-card" on:click|stopPropagation>
			<button class="close-btn" on:click={() => (showModal = false)}>
				<X size={18} />
			</button>
			<div class="modal-logo">CHASE</div>
			<h2 class="modal-title">Welcome back</h2>
			<p class="modal-subtitle">Sign in to monitor your infrastructure</p>
			<a href="/api/login" class="login-btn">
				<LogIn size={16} />
				Continue with OAuth
			</a>
		</div>
	</div>
{/if}

<style>
	.navbar {
		display: flex;
		flex-direction: column;
		height: 100%;
		padding: 0 0.75rem;
	}

	.nav-brand {
		padding: 0.5rem 0.75rem 1.5rem;
	}

	.brand-text {
		font-size: 1.125rem;
		font-weight: 700;
		color: #e5e7eb;
		letter-spacing: -0.02em;
	}

	.nav-links {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: 0.625rem;
		padding: 0.5rem 0.75rem;
		background: none;
		border: none;
		border-radius: 0.375rem;
		color: #9ca3af;
		font-size: 0.8125rem;
		font-weight: 500;
		cursor: pointer;
		transition:
			color 0.15s,
			background 0.15s;
		text-align: left;
		width: 100%;
		white-space: nowrap;
	}

	.nav-link:hover {
		color: #e5e7eb;
		background: #2b2b2b;
	}

	.nav-link.active {
		color: #e5e7eb;
		background: #2b2b2b;
	}

	.nav-link.active :global(svg) {
		color: #4ade80;
	}

	.nav-footer {
		padding-top: 0.75rem;
		border-top: 1px solid #2b2b2b;
	}

	.profile-link {
		position: relative;
	}

	.profile-link :global(img),
	.profile-link :global(svg) {
		width: 20px;
		height: 20px;
		border-radius: 50%;
	}

	.badge {
		margin-left: auto;
		min-width: 1.25rem;
		padding: 0.0625rem 0.375rem;
		border-radius: 9999px;
		background: #15803d;
		color: #4ade80;
		font-size: 0.625rem;
		font-weight: 600;
		text-align: center;
	}

	.modal-backdrop {
		position: fixed;
		inset: 0;
		display: flex;
		justify-content: center;
		align-items: center;
		background: rgba(0, 0, 0, 0.6);
		backdrop-filter: blur(4px);
		z-index: 100;
	}

	.modal-card {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		width: 320px;
		padding: 2.5rem 2rem 2rem;
		background: #1a1a1a;
		border: 1px solid #2b2b2b;
		border-radius: 0.75rem;
		box-shadow: 0 24px 48px rgba(0, 0, 0, 0.4);
	}

	.close-btn {
		position: absolute;
		top: 0.75rem;
		right: 0.75rem;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		background: none;
		border: none;
		border-radius: 0.25rem;
		color: #6b7280;
		cursor: pointer;
		transition: color 0.15s, background 0.15s;
	}

	.close-btn:hover {
		color: #e5e7eb;
		background: #2b2b2b;
	}

	.modal-logo {
		font-size: 0.8125rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: #4ade80;
		margin-bottom: 0.5rem;
	}

	.modal-title {
		font-size: 1.25rem;
		font-weight: 600;
		color: #e5e7eb;
		margin: 0;
	}

	.modal-subtitle {
		font-size: 0.8125rem;
		color: #6b7280;
		margin: 0 0 1rem;
	}

	.login-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		width: 100%;
		padding: 0.625rem 1rem;
		background: #166534;
		border-radius: 0.375rem;
		color: #e5e7eb;
		font-size: 0.875rem;
		font-weight: 600;
		text-align: center;
		text-decoration: none;
		transition: background 0.15s;
	}

	.login-btn:hover {
		background: #15803d;
	}
</style>
