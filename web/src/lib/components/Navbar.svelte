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
	<dialog open class="modal">
		<div class="modal-content">
			<button class="close-btn" on:click={() => (showModal = false)}>
				<X size={20} />
			</button>
			<div class="modal-body">
				<h2>Welcome</h2>
				<p>Sign in with your provider to continue</p>
				<a href="/api/login" class="login-btn">Login with OAuth</a>
			</div>
		</div>
	</dialog>
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

	.modal {
		position: fixed;
		inset: 0;
		display: flex;
		justify-content: center;
		align-items: center;
		border: none;
		background: #1e1e1e;
		z-index: 100;
	}

	.modal-content {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
	}

	.close-btn {
		position: absolute;
		top: 0.625rem;
		right: 1.875rem;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		color: #fff;
		cursor: pointer;
	}

	.modal-body {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 1.5rem 2rem 2rem;
		border-radius: 0.5rem;
		color: #d1d5db;
	}

	.modal-body h2 {
		font-size: 1.125rem;
		font-weight: 600;
		text-align: center;
		margin-bottom: -0.625rem;
	}

	.modal-body p {
		font-size: 0.875rem;
		color: #6b7280;
		text-align: center;
		margin-bottom: 1rem;
	}

	.login-btn {
		display: block;
		padding: 0.5rem 1rem;
		background: #6b7280;
		border-radius: 0.375rem;
		color: #fff;
		font-weight: 700;
		text-align: center;
		text-decoration: none;
		transition: background-color 0.15s ease;
	}

	.login-btn:hover {
		background: #9ca3af;
	}
</style>
