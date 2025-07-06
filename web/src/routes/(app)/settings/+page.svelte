<script>
	import BatchComponent from '$lib/components/batch/BatchComponent.svelte';
	import MassImport from '$lib/components/batch/MassImport.svelte';
</script>

<div class="max-w-2xl mx-auto mt-8">
	<h2 class="text-2xl font-bold mb-4 text-foreground pl-6">Settings</h2>

	<div class="space-y-6">
		<!-- Mass Import Section -->
		<div class="p-6 card text-card-foreground rounded-lg shadow-md">
			<MassImport />
		</div>

		<!-- Batch Operations Section -->
		<div class="p-6 card text-card-foreground rounded-lg shadow-md">
			<BatchComponent />
		</div>
	</div>
</div>
