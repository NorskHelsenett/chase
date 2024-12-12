<script lang="ts">
  import type { Server } from '$lib/models';
  import { formatDistanceToNow } from 'date-fns';

  export let server: Server;

  $: nextCheckIn = formatDistanceToNow(new Date(server.next_check), { addSuffix: true });
</script>

<div class="p-4 bg-[#202020] rounded-lg">
  <h2 class="text-lg font-medium text-white mb-4">Server Configuration</h2>

  <div class="grid gap-4">
    <div class="flex items-center justify-between p-3 rounded-md bg-[#2b2b2b]">
      <span class="text-gray-400">URL</span>
      <span class="text-white font-medium">{server.url}</span>
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
        <span class="text-white font-medium">{nextCheckIn}</span>
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
        <span class={`font-medium ${server.follow_redirect ? 'text-green-400' : 'text-yellow-400'}`}>
          {server.follow_redirect ? 'Yes' : 'No'}
        </span>
      </div>
    </div>
  </div>
</div>