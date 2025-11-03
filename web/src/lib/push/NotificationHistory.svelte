<script>
	import { onMount } from 'svelte';
	import { getNotificationHistory } from './pushClient.js';
	import { Bell, CheckCircle, XCircle, AlertTriangle, Info, Clock, ExternalLink } from 'lucide-svelte';

	let history = [];
	let isLoading = true;
	let error = null;

	onMount(async () => {
		await loadHistory();
	});

	async function loadHistory() {
		isLoading = true;
		error = null;
		try {
			history = await getNotificationHistory(50);
		} catch (err) {
			error = err.message || 'Failed to load notification history';
			console.error('Failed to load history:', err);
		} finally {
			isLoading = false;
		}
	}

	function getEventIcon(eventType) {
		switch (eventType) {
			case 'server_added':
				return '➕';
			case 'server_offline':
				return '🔴';
			case 'server_online':
				return '🟢';
			case 'server_deleted':
				return '🗑️';
			case 'server_deactivated':
				return '⏸️';
			case 'scan_completed':
				return '🔍';
			case 'high_risk_found':
				return '⚠️';
			default:
				return '📢';
		}
	}

	function formatTime(dateString) {
		const date = new Date(dateString);
		const now = new Date();
		const diffMs = now - date;
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffHours < 24) return `${diffHours}h ago`;
		if (diffDays < 7) return `${diffDays}d ago`;
		return date.toLocaleDateString();
	}
</script>

<div class="p-6">
	{#if isLoading}
		<div class="text-center py-8 text-gray-400">
			<div class="animate-spin inline-block w-6 h-6 border-2 border-gray-500 border-t-gray-300 rounded-full"></div>
			<p class="mt-2">Loading notifications...</p>
		</div>
	{:else if error}
		<div class="bg-red-900/20 border border-red-800 text-red-400 px-4 py-3 rounded">
			<div class="flex items-center gap-2">
				<XCircle size={18} />
				<span>{error}</span>
			</div>
		</div>
	{:else if history.length === 0}
		<div class="bg-[#2b2b2b] rounded-lg p-8 text-center">
			<Bell size={48} class="text-gray-600 mx-auto mb-3" />
			<p class="text-gray-400">No notifications yet</p>
			<p class="text-sm text-gray-500 mt-1">You'll see your notification history here</p>
		</div>
	{:else}
		<div class="space-y-2">
			{#each history as notification}
				<div class="bg-[#2b2b2b] rounded-lg p-4 hover:bg-[#353535] transition-colors">
					<div class="flex items-start gap-3">
						<div class="text-2xl mt-1" aria-label="Event icon">
							{getEventIcon(notification.event_type)}
						</div>
						<div class="flex-1 min-w-0">
							<h4 class="text-gray-100 font-medium">{notification.title}</h4>
							<p class="text-gray-400 text-sm mt-1">{notification.body}</p>
							{#if notification.url}
								<a
									href={notification.url}
									class="inline-flex items-center gap-1 text-xs text-blue-400 hover:text-blue-300 mt-2 transition-colors"
								>
									<ExternalLink size={12} />
									<span>View Details</span>
								</a>
							{/if}
							<div class="flex items-center gap-4 mt-2 text-xs text-gray-500">
								<div class="flex items-center gap-1">
									<Clock size={12} />
									<span>{formatTime(notification.created_at)}</span>
								</div>
								{#if notification.sent}
									<div class="flex items-center gap-1 text-green-500">
										<CheckCircle size={12} />
										<span>Sent</span>
									</div>
								{:else}
									<div class="flex items-center gap-1 text-yellow-500">
										<AlertTriangle size={12} />
										<span>Pending</span>
									</div>
								{/if}
							</div>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
