<script>
	import NotificationHistory from '$lib/push/NotificationHistory.svelte';
	import PushNotificationSettings from '$lib/push/PushNotificationSettings.svelte';
</script>

<div class="flex flex-col gap-4 p-4 min-h-screen">
	<!-- Page Header -->
	<div class="mb-2">
		<h1 class="text-3xl font-bold text-gray-100 mb-2">Notifications</h1>
		<p class="text-gray-400">Manage your notification preferences and view notification history</p>
	</div>

	<!-- Two Column Layout -->
	<div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
		<!-- Left Column - Settings -->
		<div class="lg:col-span-1">
			<div class="bg-[#202020] rounded-lg overflow-hidden sticky top-4">
				<div class="px-6 py-4 border-b border-gray-700/50">
					<h2 class="text-xl font-semibold text-gray-100">Settings</h2>
				</div>
				<PushNotificationSettings />
			</div>
		</div>

		<!-- Right Column - History -->
		<div class="lg:col-span-2">
			<div class="bg-[#202020] rounded-lg overflow-hidden">
				<NotificationHistory />
			</div>
		</div>
	</div>
</div>
