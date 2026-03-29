<script>
	import { onMount } from 'svelte';
	import {
		Home,
		Settings,
		LogIn,
		LayoutDashboard,
		Grid,
		Logs,
		Share2,
		Globe,
		X
	} from 'lucide-svelte';
	import Avatar from './Avatar.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { isLoggedIn } from '$lib/auth';
	import { tooltip } from '$lib/actions/tooltip';
	import { unreadCount, loadNotifications, resetNotifications } from '$lib/push/notificationStore';

	function submenu(node) {
		function updateSubmenuPosition(event) {
			const rect = node.getBoundingClientRect();
			const submenuX = rect.left + rect.width * 1;
			const submenuY = rect.top * 1;

			const submenuEl = node.querySelector('.submenu');
			if (submenuEl) {
				submenuEl.style.left = `${submenuX}px`;
				submenuEl.style.top = `${submenuY}px`;
				submenuEl.style.position = 'fixed';
			}
		}

		node.addEventListener('mouseenter', updateSubmenuPosition);
		return {
			destroy() {
				node.removeEventListener('mouseenter', updateSubmenuPosition);
			}
		};
	}

	const routes = [
		{ path: '/', icon: Home, tooltip: 'Home', auth: false },
		{
			path: '/dashboard?active=true',
			icon: LayoutDashboard,
			auth: true,
			submenu: [
				{ path: '/dashboard?active=true', icon: Logs, tooltip: 'Dashboard View' },
				{ path: '/grid?active=true', icon: Grid, tooltip: 'Grid View' },
				{ path: '/graph', icon: Share2, tooltip: 'Graph View' },
				{ path: '/map', icon: Globe, tooltip: 'Infrastructure Map' }
			]
		},
		{ path: '/settings', icon: Settings, tooltip: 'Settings', auth: true }
	];

	let showModal = false;
	let activeSubmenu = null;

	async function handleLogin() {
		try {
			showModal = true;
		} catch (e) {
			console.error(e.message);
		}
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

		return () => {
			unsubscribe();
		};
	});

	function handleMouseEnter(route) {
		if (route.submenu) {
			activeSubmenu = route;
		}
	}

	function handleMouseLeave() {
		activeSubmenu = null;
	}

	function formatUnread(count) {
		if (!count) {
			return '';
		}
		return count > 99 ? '99+' : `${count}`;
	}

	function isActive(routePath) {
		return $page.url.pathname === routePath || $page.url.pathname + $page.url.search === routePath;
	}
</script>

<div class="navbar">
	<div class="spacer"></div>

	<div class="nav-items">
		{#each routes as route}
			{#if !route.auth || $isLoggedIn}
				<div
					class="nav-item-wrapper"
					on:mouseenter={() => handleMouseEnter(route)}
					on:mouseleave={handleMouseLeave}
					use:submenu
				>
					<button
						class="nav-btn"
						class:active={isActive(route.path)}
						on:click={() => goto(route.path)}
						use:tooltip
						data-tooltip={route.tooltip}
					>
						<svelte:component this={route.icon} size={24} />
					</button>

					{#if activeSubmenu === route && route.submenu}
						<div class="submenu">
							{#each route.submenu as subItem}
								<button
									class="submenu-btn"
									class:active={isActive(subItem.path)}
									on:click={() => goto(subItem.path)}
									use:tooltip
									data-tooltip={subItem.tooltip}
								>
									<svelte:component this={subItem.icon} size={20} />
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{/if}
		{/each}
	</div>

	{#if $isLoggedIn}
		<div class="nav-footer">
			<button
				on:click={toProfile}
				class="profile-btn"
				use:tooltip
				data-tooltip="Profile"
			>
				<Avatar />
				{#if $unreadCount > 0}
					<span class="badge">
						{formatUnread($unreadCount)}
					</span>
				{/if}
			</button>
		</div>
	{:else}
		<div class="nav-footer">
			<button
				class="nav-btn"
				use:tooltip
				data-tooltip="Log in"
				on:click={handleLogin}
			>
				<LogIn size={24} />
			</button>
		</div>
	{/if}
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
		align-items: center;
		height: 100%;
		position: relative;
	}

	.spacer {
		margin-bottom: 2rem;
	}

	.nav-items {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 2rem;
	}

	.nav-item-wrapper {
		position: relative;
	}

	.nav-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.5rem;
		background: transparent;
		border: none;
		border-radius: 9999px;
		color: #d1d5db;
		cursor: pointer;
		transition: color 0.2s ease, background-color 0.2s ease;
	}

	.nav-btn:hover {
		color: #fff;
	}

	.nav-btn.active {
		color: #fff;
		background: #2b2b2b;
	}

	.submenu {
		position: fixed;
		background: #202020;
		border-radius: 0.5rem;
		padding: 0.5rem 0.25rem;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		max-width: 42px;
		z-index: 9999;
	}

	.submenu-btn {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.5rem;
		background: transparent;
		border: none;
		border-radius: 0.25rem;
		color: #d1d5db;
		cursor: pointer;
		transition: color 0.2s ease, background-color 0.2s ease;
	}

	.submenu-btn:hover {
		color: #fff;
	}

	.submenu-btn.active {
		color: #fff;
		background: #2b2b2b;
	}

	.nav-footer {
		margin-top: 2rem;
	}

	.profile-btn {
		position: relative;
		display: inline-flex;
		background: transparent;
		border: none;
		color: #d1d5db;
		cursor: pointer;
		transition: color 0.2s ease;
	}

	.profile-btn:hover {
		color: #fff;
	}

	.badge {
		position: absolute;
		top: -0.25rem;
		right: -0.25rem;
		min-width: 1.5rem;
		padding: 0.125rem 0.375rem;
		border-radius: 9999px;
		background: #15803d;
		border: 1px solid rgba(34, 197, 94, 0.3);
		color: #4ade80;
		font-size: 0.65rem;
		font-weight: 600;
		text-align: center;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
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
