<script lang="ts">
	import type { Server } from '$lib/models';
	import { formatDistanceToNow } from 'date-fns';
	import { Globe } from 'lucide-svelte';

	export let server: Server;

	$: nextCheckIn = formatDistanceToNow(new Date(server.next_check), { addSuffix: true });

	function getLatestPingResult(server) {
		if (!server.ping_results?.length) return null;

		return server.ping_results.reduce((latest, current) => {
			const latestTime = new Date(latest.timestamp).getTime();
			const currentTime = new Date(current.timestamp).getTime();
			return currentTime > latestTime ? current : latest;
		}, server.ping_results[0]);
	}

	function getLatestStatusCode(server) {
		const latestPing = getLatestPingResult(server);
		return latestPing?.status_code ?? 'No data';
	}

	function getStatusColor(server) {
		const latestPing = getLatestPingResult(server);
		if (!latestPing) return 'text-gray-400';
		return latestPing.status_code === server.expected_status ? 'text-green-500' : 'text-red-500';
	}
</script>

<div class="p-4 bg-[#202020] rounded-lg">
	<h2 class="text-lg font-medium text-white mb-4">Server Configuration</h2>

	<div class="grid gap-4">
		<div class="flex items-center justify-between p-3 rounded-md bg-[#2b2b2b]">
			<span class="text-gray-400">URL</span>
			<span class="text-white font-medium">
				<a
					href={`https://${server.url}`}
					target="_blank"
					rel="noopener noreferrer"
					class="block text-blue-400 hover:underline break-all"
				>
					<Globe size={20} class="inline-block mr-2" />
					{server.url}
				</a>
			</span>
		</div>

		<div class="grid grid-cols-2 gap-4">
			<div class="flex items-center justify-between p-3 rounded-md bg-[#2b2b2b]">
				<span class="text-gray-400">Status</span>
				<span class={`font-medium ${server.active ? 'text-green-400' : 'text-red-400'}`}>
					{server.active ? 'Active' : 'Inactive'}
				</span>
			</div>

			<div class="flex items-center justify-between p-3 rounded-md bg-[#2b2b2b]">
				<span class="text-gray-400">Next Check</span>
				<span class="text-white font-medium">{server.active ? nextCheckIn : 'Never'}</span>
			</div>
		</div>

		<div class="grid grid-cols-2 gap-4">
			<div class="flex items-center justify-between p-3 rounded-md bg-[#2b2b2b]">
				<span class="text-gray-400">TLS Verification</span>
				<span class={`font-medium ${server.allow_insecure ? 'text-red-400' : 'text-green-400'}`}>
					{server.allow_insecure ? 'Disabled' : 'Forced'}
				</span>
			</div>

			<div class="flex items-center justify-between p-3 rounded-md bg-[#2b2b2b]">
				<span class="text-gray-400">Follow Redirects</span>
				<span
					class={`font-medium ${server.follow_redirect ? 'text-green-400' : 'text-yellow-400'}`}
				>
					{server.follow_redirect ? 'Yes' : 'No'}
				</span>
			</div>

			<div class="flex items-center justify-between p-3 rounded-md bg-[#2b2b2b]">
				<span class="text-gray-400">First seen</span>
				<span class="text-white font-medium"
					>{new Date(server.CreatedAt).toISOString().split('T')[0]}</span
				>
			</div>

			<div class="flex items-center justify-between p-3 rounded-md bg-[#2b2b2b]">
				<span class="text-gray-400">Latest Status</span>
				<span class={`font-medium ${getStatusColor(server)}`}>
					{getLatestStatusCode(server)}
				</span>
			</div>
		</div>
	</div>
</div>
