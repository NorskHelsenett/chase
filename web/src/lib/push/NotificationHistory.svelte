<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { flip } from 'svelte/animate';
	import { fade, slide } from 'svelte/transition';
	import type { Component } from 'svelte';
	import { writable } from 'svelte/store';
	import {
		notifications,
		isLoading,
		loadError,
		hasLoaded,
		unreadCount,
		loadNotifications,
		markNotificationAsRead,
		dismissNotification,
		dismissAllNotifications
	} from './notificationStore';
	import type { NotificationEvent } from './notificationStore';
	import {
		AlertTriangle,
		Bell,
		Check,
		CheckCircle,
		Clock,
		ExternalLink,
		Link as LinkIcon,
		Loader2,
		PauseCircle,
		PlusCircle,
		Power,
		PowerOff,
		RefreshCcw,
		Search,
		ShieldAlert,
		Trash2,
		XCircle
	} from 'lucide-svelte';

	type EventVisual = {
		icon: Component;
		color: string;
		background: string;
		label: string;
	};

	const eventVisuals: Record<string, EventVisual> = {
		server_added: {
			icon: PlusCircle,
			color: 'text-emerald-400',
			background: 'bg-emerald-500/10',
			label: 'Server added'
		},
		server_offline: {
			icon: PowerOff,
			color: 'text-red-400',
			background: 'bg-red-500/10',
			label: 'Server offline'
		},
		server_online: {
			icon: Power,
			color: 'text-green-400',
			background: 'bg-green-500/10',
			label: 'Server online'
		},
		server_deleted: {
			icon: Trash2,
			color: 'text-red-500',
			background: 'bg-red-500/10',
			label: 'Server removed'
		},
		server_deactivated: {
			icon: PauseCircle,
			color: 'text-orange-400',
			background: 'bg-orange-500/10',
			label: 'Server paused'
		},
		scan_completed: {
			icon: Search,
			color: 'text-indigo-400',
			background: 'bg-indigo-500/10',
			label: 'Scan completed'
		},
		high_risk_found: {
			icon: ShieldAlert,
			color: 'text-rose-400',
			background: 'bg-rose-500/10',
			label: 'High risk detected'
		}
	};

	const defaultVisual: EventVisual = {
		icon: Bell,
		color: 'text-sky-400',
		background: 'bg-sky-500/10',
		label: 'Notification'
	};

	const pending = writable(new Set<number>());
	let bulkDismissPending = false;
	let actionError: string | null = null;
	let filterType: string = 'all';

	// Computed filtered notifications
	let filteredNotifications = $derived($notifications.filter((notification) => {
		if (filterType === 'all') return true;
		return notification.eventType === filterType;
	}));

	onMount(() => {
		loadNotifications().catch((error) => {
			console.error('Failed to load notification history:', error);
		});
	});

	function getEventVisual(eventType: string): EventVisual {
		return eventVisuals[eventType] ?? defaultVisual;
	}

	function getEventTagStyle(eventType: string): { bg: string; text: string; border: string } {
		const tagStyles: Record<string, { bg: string; text: string; border: string }> = {
			server_added: {
				bg: 'bg-green-500/10',
				text: 'text-green-500',
				border: 'border-green-500/30'
			},
			server_online: {
				bg: 'bg-green-500/10',
				text: 'text-green-500',
				border: 'border-green-500/30'
			},
			server_offline: {
				bg: 'bg-amber-500/10',
				text: 'text-amber-400',
				border: 'border-amber-500/30'
			},
			server_deactivated: {
				bg: 'bg-amber-500/10',
				text: 'text-amber-400',
				border: 'border-amber-500/30'
			},
			server_deleted: { bg: 'bg-red-500/10', text: 'text-red-500', border: 'border-red-500/30' },
			scan_completed: { bg: 'bg-blue-500/10', text: 'text-blue-500', border: 'border-blue-500/30' },
			high_risk_found: { bg: 'bg-red-500/10', text: 'text-red-500', border: 'border-red-500/30' }
		};
		return (
			tagStyles[eventType] ?? {
				bg: 'bg-gray-700/50',
				text: 'text-gray-300',
				border: 'border-gray-600/30'
			}
		);
	}

	function getEventLabel(eventType: string): string {
		const labels: Record<string, string> = {
			server_added: 'New',
			server_online: 'Online',
			server_offline: 'Offline',
			server_deleted: 'Deleted',
			server_deactivated: 'Deactivated',
			scan_completed: 'Scan',
			high_risk_found: 'Alert'
		};
		return labels[eventType] ?? 'New';
	}

	function formatTime(dateString: string | null | undefined): string {
		if (!dateString) {
			return '';
		}

		const date = new Date(dateString);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffHours < 24) return `${diffHours}h ago`;
		if (diffDays < 7) return `${diffDays}d ago`;
		return date.toLocaleDateString();
	}

	function beginPending(id: number) {
		pending.update((set) => {
			const next = new Set(set);
			next.add(id);
			return next;
		});
	}

	function endPending(id: number) {
		pending.update((set) => {
			const next = new Set(set);
			next.delete(id);
			return next;
		});
	}

	async function handleOpen(notification: NotificationEvent) {
		// Don't navigate if server is deleted
		if (notification.eventType === 'server_deleted') {
			return;
		}

		if (!notification.serverId) {
			console.warn('Cannot open notification: missing server ID');
			return;
		}

		// Mark as read
		if (!notification.read) {
			markNotificationAsRead(notification.id, { skipRequest: false }).catch(() => {
				console.warn('Failed to mark notification as read');
			});
		}

		// Navigate to server page
		await goto(`/server/${notification.serverId}`);
	}

	async function handleMarkRead(notification: NotificationEvent) {
		actionError = null;
		beginPending(notification.id);
		try {
			await markNotificationAsRead(notification.id);
		} catch (error) {
			actionError = error instanceof Error ? error.message : 'Failed to mark notification as read';
		} finally {
			endPending(notification.id);
		}
	}

	async function handleDismiss(notification: NotificationEvent) {
		actionError = null;
		beginPending(notification.id);
		try {
			await dismissNotification(notification.id);
		} catch (error) {
			actionError = error instanceof Error ? error.message : 'Failed to dismiss notification';
		} finally {
			endPending(notification.id);
		}
	}

	async function handleDismissAll() {
		if (!$notifications.length) {
			return;
		}

		actionError = null;
		bulkDismissPending = true;
		try {
			await dismissAllNotifications();
		} catch (error) {
			actionError = error instanceof Error ? error.message : 'Failed to dismiss notifications';
		} finally {
			bulkDismissPending = false;
		}
	}

	async function handleRefresh() {
		actionError = null;
		try {
			await loadNotifications(true);
		} catch (error) {
			actionError = error instanceof Error ? error.message : 'Failed to refresh notifications';
		}
	}
</script>

<div class="p-6">
	<div class="flex flex-wrap items-center justify-between gap-4 mb-4">
	<!-- Filter Buttons -->
	<div class="flex flex-wrap items-center gap-2">
		<button
			class={`px-3 py-1.5 text-xs rounded-lg transition-colors ${
				filterType === 'all'
					? 'bg-gray-700/50 text-gray-200'
					: 'bg-gray-800/30 text-gray-400 hover:bg-gray-700/30 hover:text-gray-300'
			}`}
			onclick={() => (filterType = 'all')}
		>
			All
		</button>
		<button
			class={`px-3 py-1.5 text-xs rounded-lg transition-colors ${
				filterType === 'server_added'
					? 'bg-green-500/20 text-green-400 border border-green-500/30'
					: 'bg-gray-800/30 text-gray-400 hover:bg-green-500/10 hover:text-green-400'
			}`}
			onclick={() => (filterType = 'server_added')}
		>
			New
		</button>
		<button
			class={`px-3 py-1.5 text-xs rounded-lg transition-colors ${
				filterType === 'server_online'
					? 'bg-green-500/20 text-green-400 border border-green-500/30'
					: 'bg-gray-800/30 text-gray-400 hover:bg-green-500/10 hover:text-green-400'
			}`}
			onclick={() => (filterType = 'server_online')}
		>
			Online
		</button>
		<button
			class={`px-3 py-1.5 text-xs rounded-lg transition-colors ${
				filterType === 'server_offline'
					? 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
					: 'bg-gray-800/30 text-gray-400 hover:bg-amber-500/10 hover:text-amber-400'
			}`}
			onclick={() => (filterType = 'server_offline')}
		>
			Offline
		</button>
		<button
			class={`px-3 py-1.5 text-xs rounded-lg transition-colors ${
				filterType === 'server_deleted'
					? 'bg-red-500/20 text-red-500 border border-red-500/30'
					: 'bg-gray-800/30 text-gray-400 hover:bg-red-500/10 hover:text-red-500'
			}`}
			onclick={() => (filterType = 'server_deleted')}
		>
			Deleted
		</button>
	</div>
		<div class="flex flex-wrap items-center gap-2">
			<button
				class="flex items-center gap-1.5 rounded-lg border border-gray-700/50 px-3 py-1.5 text-sm text-gray-300 transition-colors hover:border-gray-600 hover:bg-gray-800/30 disabled:opacity-50"
				onclick={handleRefresh}
				disabled={$isLoading}
			>
				{#if $isLoading}
					<Loader2 size={14} class="animate-spin" />
				{:else}
					<RefreshCcw size={14} />
				{/if}
				<span>Refresh</span>
			</button>
			<button
				class="flex items-center gap-1.5 rounded-lg bg-gray-700/30 px-3 py-1.5 text-sm text-gray-200 transition-colors hover:bg-gray-700/50 disabled:opacity-50"
				onclick={handleDismissAll}
				disabled={bulkDismissPending || !$notifications.length}
			>
				{#if bulkDismissPending}
					<Loader2 size={14} class="animate-spin" />
				{:else}
					<XCircle size={14} />
				{/if}
				<span>Dismiss all</span>
			</button>
		</div>
	</div>

	{#if $isLoading && !$hasLoaded}
		<div class="flex flex-col gap-3">
			{#each Array(3) as _, index}
				<div
					class="animate-pulse rounded-lg border border-gray-700/50 bg-[#2b2b2b] p-4"
					aria-hidden="true"
				>
					<div class="flex items-start gap-3">
						<div class="h-10 w-10 rounded-lg bg-gray-700/60"></div>
						<div class="flex-1 space-y-3">
							<div class="h-4 w-48 rounded bg-gray-700/60"></div>
							<div class="h-3 w-64 rounded bg-gray-700/40"></div>
							<div class="h-3 w-32 rounded bg-gray-700/40"></div>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if $loadError}
		<div class="rounded-lg border border-red-700/50 bg-red-900/10 p-4 text-red-300">
			<div class="flex items-center gap-2">
				<XCircle size={18} />
				<span class="text-sm">{$loadError}</span>
			</div>
			<button
				class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-red-500/40 px-3 py-1.5 text-sm text-red-200 transition-colors hover:border-red-400 hover:bg-red-900/20"
				onclick={handleRefresh}
			>
				<RefreshCcw size={14} />
				<span>Try again</span>
			</button>
		</div>
	{:else if !$notifications.length}
		<div class="rounded-lg border border-dashed border-gray-700/50 bg-[#2b2b2b] p-12 text-center">
			<Bell size={40} class="mx-auto text-gray-600" />
			<p class="mt-4 text-gray-300 font-medium">No notifications yet</p>
			<p class="text-sm text-muted-foreground mt-2">
				Notification history is only created when you have active push notification subscriptions.
			</p>
			<p class="text-sm text-muted-foreground mt-1">
				Enable push notifications in the settings panel to start receiving and tracking notification
				events.
			</p>
		</div>
	{:else if filteredNotifications.length === 0}
		<div class="rounded-lg border border-dashed border-gray-700/50 bg-[#2b2b2b] p-12 text-center">
			<Bell size={40} class="mx-auto text-gray-600" />
			<p class="mt-4 text-gray-300 font-medium">No notifications found</p>
			<p class="text-sm text-muted-foreground mt-2">
				Try selecting a different filter to see more notifications.
			</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each filteredNotifications as notification (notification.id ?? `${notification.eventType}-${notification.createdAt}-${notification.title}`)}
				{@const visual = getEventVisual(notification.eventType)}
				{@const tagStyle = getEventTagStyle(notification.eventType)}
				{@const eventLabel = getEventLabel(notification.eventType)}
				{@const isDeleted = notification.eventType === 'server_deleted'}
				<div
					in:fade={{ duration: 200 }}
					out:fade={{ duration: 150 }}
					animate:flip={{ duration: 300 }}
				>
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class={`group relative overflow-hidden rounded-lg p-4 transition-all w-full text-left ${
							isDeleted ? 'cursor-default' : 'cursor-pointer'
						} ${
							notification.read
								? 'border-gray-800/50 bg-[#1e1e1e] hover:border-gray-700/60'
								: 'border-gray-600/50 bg-[#2b2b2b] hover:border-gray-500/60'
						}`}
						role="button"
						tabindex="0"
						onclick={() => handleOpen(notification)}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleOpen(notification); } }}
					>
						<div class="flex items-start gap-3">
							<div
								class={`flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg ${notification.read ? 'opacity-50' : ''} ${visual.background} ${visual.color}`}
								aria-label={visual.label}
							>
								<visual.icon size={18} />
							</div>
							<div class="flex-1 min-w-0">
								<div class="flex flex-wrap items-start justify-between gap-2 mb-2">
									<div class="flex-1">
										<h4
											class={`text-base font-semibold ${notification.read ? 'text-gray-500' : 'text-gray-100'}`}
										>
											{notification.title}
										</h4>
										<p
											class={`mt-1 text-sm ${notification.read ? 'text-gray-600' : 'text-muted-foreground'}`}
										>
											{notification.body}
										</p>
									</div>
									<span
										class="rounded-full px-2 py-0.5 text-xs font-medium {tagStyle.bg} {tagStyle.text} border {tagStyle.border}"
									>
										{eventLabel}
									</span>
								</div>

								{#if notification.serverUrl}
									<div class="mt-3 flex items-center gap-1.5 text-sm text-muted-foreground">
										<LinkIcon size={14} />
										<span class="truncate">{notification.serverUrl}</span>
									</div>
								{/if}

								<div class="mt-3 flex flex-wrap items-center gap-3 text-xs text-gray-500">
									<div class="flex items-center gap-1">
										<Clock size={11} />
										<span>{formatTime(notification.createdAt)}</span>
									</div>
									{#if notification.readAt}
										<div class="flex items-center gap-1">
											<Check size={11} />
											<span>Read {formatTime(notification.readAt)}</span>
										</div>
									{/if}
									{#if notification.sent}
										<div class="flex items-center gap-1 text-emerald-500">
											<CheckCircle size={11} />
											<span>Delivered</span>
										</div>
									{:else}
										<div class="flex items-center gap-1 text-amber-500">
											<AlertTriangle size={11} />
											<span>Pending</span>
										</div>
									{/if}
								</div>

								<div class="mt-4 flex flex-wrap items-center gap-2">
									{#if !notification.read}
										<button
											class="inline-flex items-center gap-1.5 rounded-lg border border-gray-700/50 px-3 py-1.5 text-sm text-gray-300 transition-colors hover:border-gray-600 hover:bg-[#3a3a3a] disabled:opacity-50"
											onclick={(e) => {
												e.stopPropagation();
												handleMarkRead(notification);
											}}
											disabled={$pending.has(notification.id)}
										>
											{#if $pending.has(notification.id)}
												<Loader2 size={14} class="animate-spin" />
											{:else}
												<Check size={14} />
											{/if}
											<span>Mark read</span>
										</button>
									{/if}

									<button
										class="inline-flex items-center gap-1.5 rounded-lg border border-transparent px-3 py-1.5 text-sm text-gray-400 transition-colors hover:border-red-600/50 hover:bg-red-900/10 hover:text-red-400 disabled:opacity-50"
										onclick={(e) => {
											e.stopPropagation();
											handleDismiss(notification);
										}}
										disabled={$pending.has(notification.id)}
									>
										{#if $pending.has(notification.id)}
											<Loader2 size={14} class="animate-spin" />
										{:else}
											<XCircle size={14} />
										{/if}
										<span>Dismiss</span>
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	{#if actionError}
		<div class="rounded-lg border border-red-700/50 bg-red-900/10 p-3 text-sm text-red-300">
			{actionError}
		</div>
	{/if}
</div>
