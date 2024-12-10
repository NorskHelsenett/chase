<script>
  import { onMount } from 'svelte';
  import { isLoggedIn, user } from '$lib/auth';
  import { goto } from '$app/navigation';
  import { User, Mail, Settings, Edit2 } from 'lucide-svelte';
  import Avatar from '$lib/components/Avatar.svelte';

  onMount(() => {
    const unsubscribe = isLoggedIn.subscribe(loggedIn => {
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
              <Avatar />
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

          <!-- Actions Section -->
          <div class="mt-8 pt-6 border-t border-secondary">
            <button class="inline-flex items-center px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-100 rounded-lg transition-colors duration-150">
              <Edit2 class="w-5 h-5 mr-2" />
              Edit Profile
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}