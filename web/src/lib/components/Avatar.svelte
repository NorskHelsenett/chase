<script lang="ts">
	import { user, initializeAuth, getEmailHash } from '$lib/auth';
	import { onMount } from 'svelte';
	import type { AvatarSize } from '../types/avatarSizes';

	interface Props {
		size?: AvatarSize;
	}

	let { size = 'small' }: Props = $props();

	// Size mappings object
	const sizeMap: Record<AvatarSize, string> = {
		small: 'w-8 h-8',
		medium: 'w-12 h-12',
		large: 'w-16 h-16'
	};

	onMount(async () => {
		await initializeAuth();
	});

	function getGravatarUrl(email: string): string {
		const hash = getEmailHash(email);
		return `https://www.gravatar.com/avatar/${hash}?d=mp`;
	}
</script>

{#if $user}
	<img src={getGravatarUrl($user.email)} alt="User avatar" class="{sizeMap[size]} rounded-full" />
{/if}
