<script>
	import { user, initializeAuth, getEmailHash } from '$lib/auth';
	import { onMount } from 'svelte';

	onMount(async () => {
		await initializeAuth();
	});

	function getGravatarUrl(email) {
		const hash = getEmailHash(email);
		return `https://www.gravatar.com/avatar/${hash}?d=mp`;
	}
</script>

{#if $user}
	<img src={getGravatarUrl($user.email)} alt="User avatar" class="w-8 h-8 rounded-full" />
{/if}
