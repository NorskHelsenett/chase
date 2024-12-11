<script>
	import { Home, Settings, LogIn } from 'lucide-svelte';
	import Avatar from './Avatar.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { isLoggedIn } from '$lib/auth'
	import { tooltip } from '$lib/actions/tooltip';

	const routes = [
    { path: '/', icon: Home, tooltip: 'Home' },
    { path: '/settings', icon: Settings, tooltip: 'Settings' }
  ];

	let showModal = false;

	async function handleLogin() {
   try {
     showModal = true;
   } catch (e) {
     console.error(e.message);
   }
 }

 async function toProfile() {
	await goto("/profile")
 }

</script>

<div class="flex flex-col items-center h-full">
	<!-- Top icon -->
	<div class="mb-8">

	</div>

	<!-- Center icons -->
	<div class="flex-grow flex flex-col items-center justify-center space-y-8">
		{#each routes as route}
			<button
				class="text-gray-300 hover:text-white transition-colors duration-200 ease-in-out p-2 tooltip"
				class:text-white={$page.url.pathname === route.path}
				class:bg-gray-700={$page.url.pathname === route.path}
				class:rounded-full={$page.url.pathname === route.path}
				on:click={() => goto(route.path)}
				use:tooltip
				data-tooltip={route.tooltip}
			>
				<svelte:component this={route.icon} size={24} />
			</button>
		{/each}
	</div>

	{#if $isLoggedIn}
	<!-- Bottom icon -->
	<div class="mt-8">
		<button on:click={toProfile} class="text-gray-300 hover:text-white tooltip" use:tooltip data-tooltip="Profile">
			<Avatar />
		</button>
		<!-- <button class="text-gray-300 hover:text-white p-2 tooltip" data-tooltip="Log out" on:click={handleLogout} use:tooltip >
			<LogOut size={24} />
		</button> -->
	</div>
	{:else}
	<div class="mt-8">
		<button use:tooltip class="text-gray-300 hover:text-white p-2 tooltip" data-tooltip="Log in" on:click={handleLogin}>
			<LogIn size={24} />
		</button>
	</div>
	{/if}
</div>

{#if showModal}
 <dialog open class="modal fixed inset-0 overflow-y-auto h-full w-full">
   <div class="flex items-center justify-center min-h-screen">
		 <button class="close-button self-end justify-self-end" on:click={() => (showModal = false)}>×</button>
     <div class="rounded px-8 pt-6 pb-8 mb-4 text-gray-300 grid gap-2">
       <h2 class="text-lg font-semibold text-center">Welcome</h2>
       <p class="text-sm text-gray-500 mb-4 text-center">Sign in with your provider to continue</p>
       <a href="/api/login" class="bg-gray-500 hover:bg-gray-400 text-white font-bold py-2 px-4 rounded text-center">Login with OAuth</a>
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

	h2{
		margin-bottom: -10px;
	}

	.close-button {
		position: absolute;
		top: 10px;
		right: 30px;
		font-size: 24px;
		color:white;
		background-color: transparent;
		border: none;
		cursor: pointer;
	}
 </style>