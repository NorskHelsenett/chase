<script>
	import {
		Home,
		Settings,
		LogIn,
		LayoutDashboard,
		Grid,
		Layout,
		Logs,
		Share2
	} from 'lucide-svelte';
	import Avatar from './Avatar.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { isLoggedIn } from '$lib/auth';
	import { tooltip } from '$lib/actions/tooltip';

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
				{ path: '/graph', icon: Share2, tooltip: 'Graph View' }
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

	function handleMouseEnter(route) {
		if (route.submenu) {
			activeSubmenu = route;
		}
	}

	function handleMouseLeave() {
		activeSubmenu = null;
	}
</script>

<div class="flex flex-col items-center h-full relative">
	<!-- Top icon -->
	<div class="mb-8"></div>

	<!-- Center icons -->
	<div class="flex-grow flex flex-col items-center justify-center space-y-8">
		{#each routes as route}
			{#if !route.auth || $isLoggedIn}
				<div
					class="relative"
					on:mouseenter={() => handleMouseEnter(route)}
					on:mouseleave={handleMouseLeave}
					use:submenu
				>
					<button
						class="text-gray-300 hover:text-white transition-colors duration-200 ease-in-out p-2 tooltip"
						class:text-white={$page.url.pathname === route.path}
						class:bg-[#2b2b2b]={$page.url.pathname === route.path}
						class:rounded-full={$page.url.pathname === route.path}
						on:click={() => goto(route.path)}
						use:tooltip
						data-tooltip={route.tooltip}
					>
						<svelte:component this={route.icon} size={24} />
					</button>

					{#if activeSubmenu === route && route.submenu}
						<div
							class="submenu bg-[#202020] rounded-lg shadow-lg py-2 px-1 fixed"
							style="max-width: 42px; z-index: 9999;"
						>
							{#each route.submenu as subItem}
								<button
									class="w-full text-gray-300 hover:text-white transition-colors duration-200 ease-in-out p-2 tooltip flex justify-center"
									class:text-white={$page.url.pathname === subItem.path}
									class:bg-[#2b2b2b]={$page.url.pathname === subItem.path}
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
		<!-- Bottom icon -->
		<div class="mt-8">
			<button
				on:click={toProfile}
				class="text-gray-300 hover:text-white tooltip"
				use:tooltip
				data-tooltip="Profile"
			>
				<Avatar />
			</button>
		</div>
	{:else}
		<div class="mt-8">
			<button
				use:tooltip
				class="text-gray-300 hover:text-white p-2 tooltip"
				data-tooltip="Log in"
				on:click={handleLogin}
			>
				<LogIn size={24} />
			</button>
		</div>
	{/if}
</div>

{#if showModal}
	<dialog open class="modal fixed inset-0 overflow-y-auto h-full w-full">
		<div class="flex items-center justify-center min-h-screen">
			<button class="close-button self-end justify-self-end" on:click={() => (showModal = false)}
				>×</button
			>
			<div class="rounded px-8 pt-6 pb-8 mb-4 text-gray-300 grid gap-2">
				<h2 class="text-lg font-semibold text-center">Welcome</h2>
				<p class="text-sm text-gray-500 mb-4 text-center">Sign in with your provider to continue</p>
				<a
					href="/api/login"
					class="bg-gray-500 hover:bg-gray-400 text-white font-bold py-2 px-4 rounded text-center"
					>Login with OAuth</a
				>
			</div>
		</div>
	</dialog>
{/if}

<style>
	.modal {
		position: fixed;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		display: flex;
		justify-content: center;
		align-items: center;
		border: none;
		background-color: rgba(30, 30, 30, 1);
	}

	h2 {
		margin-bottom: -10px;
	}

	.close-button {
		position: absolute;
		top: 10px;
		right: 30px;
		font-size: 24px;
		color: white;
		background-color: transparent;
		border: none;
		cursor: pointer;
	}
</style>
