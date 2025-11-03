<script>
	import { onMount } from 'svelte';
	import { isLoggedIn, user } from '$lib/auth';
	import { goto } from '$app/navigation';
	import { User, Mail, ChevronDown, ChevronUp } from 'lucide-svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import ApiToken from '$lib/components/ApiToken.svelte';
	import PushNotificationSettings from '$lib/push/PushNotificationSettings.svelte';
	import NotificationHistory from '$lib/push/NotificationHistory.svelte';

	let pushSettingsOpen = false;
	let notificationHistoryOpen = false;

	onMount(() => {
		const unsubscribe = isLoggedIn.subscribe((loggedIn) => {
			if (!loggedIn) {
				goto('/');
			}
		});

		return () => {
			unsubscribe();
		};
	});
</script>

{#if $isLoggedIn}
	<div class="flex-1 p-8">
		<div class="max-w-4xl mx-auto">
			<!-- Header -->
			<div class="mb-8">
				<h1 class="text-3xl font-bold text-gray-100 mb-2">Profile</h1>
				<p class="text-gray-400">Manage your account settings and preferences</p>
			</div>

			<!-- Profile Content -->
			{#if $user}
				<div class="bg-secondary rounded-lg p-6 shadow-lg">
					<!-- Avatar Section -->
					<div class="flex items-center space-x-6 mb-8 pb-6 border-b border-secondary">
						<div class="w-20 h-20">
							<Avatar size="large" />
						</div>
						<div>
							<h2 class="text-xl font-semibold text-gray-100">{$user.name}</h2>
							<p class="text-gray-400">Member</p>
						</div>
					</div>

					<!-- User Info Section -->
					<div class="space-y-6">
						<!-- Name -->
						<div class="flex items-center space-x-4">
							<div class="bg-gray-700 p-3 rounded-lg">
								<User class="w-6 h-6 text-gray-300" />
							</div>
							<div>
								<p class="text-sm text-gray-400">Name</p>
								<p class="text-lg text-gray-100">{$user.name}</p>
							</div>
						</div>

						<!-- Email -->
						<div class="flex items-center space-x-4">
							<div class="bg-gray-700 p-3 rounded-lg">
								<Mail class="w-6 h-6 text-gray-300" />
							</div>
							<div>
								<p class="text-sm text-gray-400">Email</p>
								<p class="text-lg text-gray-100">{$user.email}</p>
							</div>
						</div>
					</div>

					<ApiToken />

				</div>
				
				<!-- Push Notification Settings -->
				<div class="mt-6">
					<div class="bg-[#202020] rounded-lg overflow-hidden">
						<button
							on:click={() => (pushSettingsOpen = !pushSettingsOpen)}
							class="w-full px-6 py-4 flex items-center justify-between hover:bg-[#252525] transition-colors"
						>
							<h2 class="text-xl text-gray-100">Push Notifications</h2>
							{#if pushSettingsOpen}
								<ChevronUp size={20} class="text-gray-400" />
							{:else}
								<ChevronDown size={20} class="text-gray-400" />
							{/if}
						</button>
						{#if pushSettingsOpen}
							<div class="">
								<PushNotificationSettings />
							</div>
						{/if}
					</div>
				</div>

				<!-- Notification History -->
				<div class="mt-6">
					<div class="bg-[#202020] rounded-lg overflow-hidden">
						<button
							on:click={() => (notificationHistoryOpen = !notificationHistoryOpen)}
							class="w-full px-6 py-4 flex items-center justify-between hover:bg-[#252525] transition-colors"
						>
							<h2 class="text-xl text-gray-100">Notification History</h2>
							{#if notificationHistoryOpen}
								<ChevronUp size={20} class="text-gray-400" />
							{:else}
								<ChevronDown size={20} class="text-gray-400" />
							{/if}
						</button>
						{#if notificationHistoryOpen}
							<div class="">
								<NotificationHistory />
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}
