<script>
	import { onMount } from 'svelte';
	import { Bell, BellOff, Check, X, AlertCircle, Loader } from 'lucide-svelte';
	import CustomCheckbox from '$lib/components/ui/CustomCheckbox.svelte';
	import {
		initPushNotifications,
		subscribeToPush,
		unsubscribeFromPush,
		isPushSubscribed,
		getNotificationPreferences,
		updateNotificationPreferences,
		getEventTypes,
		sendTestNotification
	} from './pushClient.js';

	let isSupported = false;
	let isSubscribed = false;
	let isLoading = false;
	let permission = 'default';
	let preferences = {};
	let eventTypes = [];
	let error = null;
	let successMessage = null;

	onMount(async () => {
		isSupported = 'serviceWorker' in navigator && 'PushManager' in window;
		if (!isSupported) return;

		if ('Notification' in window) {
			permission = Notification.permission;
		}

		await initPushNotifications();
		isSubscribed = await isPushSubscribed();

		try {
			eventTypes = await getEventTypes();
		} catch (err) {
			console.error('Failed to load event types:', err);
		}

		if (isSubscribed) {
			await loadPreferences();
		}
	});

	async function loadPreferences() {
		try {
			preferences = await getNotificationPreferences();
		} catch (err) {
			console.error('Failed to load preferences:', err);
		}
	}

	async function handleSubscribe() {
		isLoading = true;
		error = null;
		successMessage = null;

		try {
			await subscribeToPush();
			isSubscribed = true;
			permission = 'granted';
			successMessage = 'Successfully subscribed to notifications!';
			await loadPreferences();
		} catch (err) {
			error = err.message || 'Failed to subscribe to notifications';
			console.error('Subscribe error:', err);
		} finally {
			isLoading = false;
		}
	}

	async function handleUnsubscribe() {
		isLoading = true;
		error = null;
		successMessage = null;

		try {
			await unsubscribeFromPush();
			isSubscribed = false;
			successMessage = 'Successfully unsubscribed from notifications';
		} catch (err) {
			error = err.message || 'Failed to unsubscribe';
			console.error('Unsubscribe error:', err);
		} finally {
			isLoading = false;
		}
	}

	async function handlePreferenceChange(eventType) {
		error = null;
		try {
			const newPreferences = { ...preferences };
			newPreferences[eventType] = !newPreferences[eventType];
			await updateNotificationPreferences(newPreferences);
			preferences = newPreferences;
		} catch (err) {
			error = err.message || 'Failed to update preferences';
			console.error('Preference update error:', err);
		}
	}

	async function handleTestNotification() {
		isLoading = true;
		error = null;
		successMessage = null;

		try {
			await sendTestNotification();
			successMessage = 'Test notification sent! Check your notifications.';
		} catch (err) {
			error = err.message || 'Failed to send test notification';
			console.error('Test notification error:', err);
		} finally {
			isLoading = false;
		}
	}

	function clearMessages() {
		error = null;
		successMessage = null;
	}
</script>

<div class="p-6">
	{#if !isSupported}
		<div class="bg-yellow-900/20 border border-yellow-800 text-yellow-400 px-4 py-3 rounded-lg">
			<div class="flex items-center gap-2">
				<AlertCircle size={18} />
				<span>Push notifications are not supported in your browser</span>
			</div>
		</div>
	{:else if permission === 'denied'}
		<div class="bg-red-900/20 border border-red-800 text-red-400 px-4 py-3 rounded-lg">
			<div class="flex items-center gap-2">
				<X size={18} />
				<span>Notifications are blocked. Please enable them in your browser settings</span>
			</div>
		</div>
	{:else}
		{#if error}
			<div
				class="bg-red-900/20 border border-red-800 text-red-400 px-4 py-3 rounded-lg mb-4 cursor-pointer hover:bg-red-900/30 transition-colors"
				on:click={clearMessages}
			>
				<div class="flex items-center gap-2">
					<X size={18} />
					<span>{error}</span>
				</div>
			</div>
		{/if}

		{#if successMessage}
			<div
				class="bg-green-900/20 border border-green-800 text-green-400 px-4 py-3 rounded-lg mb-4 cursor-pointer hover:bg-green-900/30 transition-colors"
				on:click={clearMessages}
			>
				<div class="flex items-center gap-2">
					<Check size={18} />
					<span>{successMessage}</span>
				</div>
			</div>
		{/if}

		{#if !isSubscribed}
			<div class="mb-4">
				<p class="text-gray-400 text-sm mb-4">
					Enable push notifications to receive real-time alerts about server status changes and
					security events
				</p>
				<button
					on:click={handleSubscribe}
					disabled={isLoading}
					class="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white flex items-center gap-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{#if isLoading}
						<Loader size={18} class="animate-spin" />
						<span>Subscribing...</span>
					{:else}
						<Bell size={18} />
						<span>Enable Notifications</span>
					{/if}
				</button>
			</div>
		{:else}
			<div class="space-y-4">
				<div class="flex items-center justify-between p-3 bg-green-900/10 border border-green-800/30 rounded-lg">
					<div class="flex items-center gap-2 text-green-400">
						<Check size={18} />
						<span class="text-sm">Notifications Enabled</span>
					</div>
					<button
						on:click={handleUnsubscribe}
						disabled={isLoading}
						class="text-sm text-red-400 hover:text-red-300 disabled:text-gray-500 transition-colors"
					>
						Unsubscribe
					</button>
				</div>

				<div>
					<h4 class="text-gray-100 font-medium mb-3">Notification Preferences</h4>
					<div class="space-y-2">
						{#each eventTypes as eventType}
							<div class="p-3 bg-[#2b2b2b] rounded-lg transition-colors">
								<CustomCheckbox
									checked={preferences[eventType.type] !== false}
									on:change={() => handlePreferenceChange(eventType.type)}
									label={eventType.name || eventType.type}
								/>
								<p class="text-gray-400 text-xs ml-7 mt-1">{eventType.description || ''}</p>
							</div>
						{/each}
					</div>
				</div>

				<div class="pt-4">
					<button
						on:click={handleTestNotification}
						disabled={isLoading}
						class="bg-[#2b2b2b] hover:bg-[#353535] disabled:bg-gray-800 disabled:text-gray-500 text-gray-100 px-4 py-2 rounded-lg flex items-center gap-2 transition-colors text-sm"
					>
						{#if isLoading}
							<Loader size={16} class="animate-spin" />
							<span>Sending...</span>
						{:else}
							<Bell size={16} />
							<span>Send Test Notification</span>
						{/if}
					</button>
				</div>
			</div>
		{/if}
	{/if}
</div>
