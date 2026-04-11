<script>
	import MassImport from '$lib/components/batch/MassImport.svelte';
	import JobScheduler from '$lib/components/settings/JobScheduler.svelte';
</script>

<div class="flex-1 p-8">
	<div class="max-w-7xl xl:max-w-[100rem] mx-auto">
		<div class="mb-8">
			<h1 class="text-3xl font-bold text-gray-100 mb-2">Settings</h1>
			<p class="text-gray-400">System overview and job management</p>
		</div>

		<JobScheduler />

		<div class="mt-6">
			<div class="bg-[#202020] rounded-lg p-6">
				<MassImport />
			</div>
		</div>
	</div>
</div>
