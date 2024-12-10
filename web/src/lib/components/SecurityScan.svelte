<script>
	import { Lock, Wrench, AlertTriangle, Server, Book } from 'lucide-svelte';
	import { fade } from 'svelte/transition';
	import { onMount } from 'svelte';

	/** @type {string} */
	export let domain = '';

	let loading = true;
	let results = null;
	let error = null;

	async function performScan() {
		loading = true;
		error = null;

		try {
			const response = await fetch(`/api/security/${encodeURIComponent(domain)}`);
			if (!response.ok) throw new Error('Scan failed');
			results = await response.json();
		} catch (err) {
			error = err.message;
			console.error('Scan failed:', err);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		performScan();
	});
</script>

<div class="min-h-screen w-full text-gray-100">
	{#if error}
		<div class="max-w-3xl mx-auto p-4">
			<div class="bg-red-900/20 border border-red-900 rounded-lg p-4 text-red-400">
				{error}
			</div>
		</div>
	{/if}

	<section class="mb-8">
		{#if loading}
			<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
				<div class="h-48 bg-gray-700 rounded-lg w-full mb-4"></div>
				<div class="space-y-2">
					{#each Array(3) as _}
						<div class="h-4 bg-gray-700 rounded w-full"></div>
					{/each}
				</div>
			</div>
		{:else if results}
			<div class="bg-[#202020] rounded-lg overflow-hidden" in:fade={{ duration: 200 }}>
				<!-- Website Screenshot -->
				<div class="w-full h-48 bg-[#2b2b2b] relative">
					<img
						src="/api/placeholder/800/400"
						alt="Website preview"
						class="w-full h-full object-cover opacity-80"
					/>
					<div class="absolute bottom-0 left-0 right-0 h-16"></div>
				</div>

				<!-- Summary Content -->
				<div class="p-6">
					<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
						<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
							<div
								class="text-2xl font-bold mb-1"
								class:text-green-500={results.headers.score === 'A+'}
								class:text-yellow-500={results.headers.score === 'B+'}
							>
								{results.headers.score}
							</div>
							<div class="text-sm text-gray-400">Headers</div>
						</div>
						<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
							<div
								class="text-2xl font-bold mb-1"
								class:text-green-500={results.certificate.grade === 'A'}
							>
								{results.certificate.grade}
							</div>
							<div class="text-sm text-gray-400">SSL/TLS</div>
						</div>
						<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
							<div class="text-2xl font-bold mb-1 uppercase text-yellow-500">
								{results.adminPages.risk}
							</div>
							<div class="text-sm text-gray-400">Admin Risk</div>
						</div>
						<div class="text-center p-3 bg-[#2b2b2b] rounded-lg">
							<div class="text-2xl font-bold mb-1 uppercase text-red-500">
								{results.swagger.risk}
							</div>
							<div class="text-sm text-gray-400">API Risk</div>
						</div>
					</div>

					<div class="space-y-2">
						<div class="text-gray-400">Key Findings:</div>
						<ul class="text-gray-300 space-y-1">
							<li>{results.headers.issues[0]}</li>
							<li>Certificate valid until {results.certificate.validUntil}</li>
							<li>{results.adminPages.exposed.length} admin pages exposed</li>
							<li>
								{results.swagger.exposed
									? 'API documentation publicly accessible'
									: 'No public API documentation'}
							</li>
						</ul>
					</div>
				</div>
			</div>
		{/if}
	</section>

	<div class="max-w-3xl mx-auto p-4 space-y-6">
		<!-- Security Headers Section -->
		<section>
			<h2 class="text-xl flex items-center gap-2 mb-4">
				<Lock class="w-5 h-5" />
				Security Headers Analysis
			</h2>

			{#if loading}
				<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
					<div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
					<div class="space-y-2">
						{#each Array(3) as _}
							<div class="h-4 bg-gray-700 rounded w-full"></div>
						{/each}
					</div>
				</div>
			{:else if results}
				<div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
					<div class="flex items-center gap-4 mb-6">
						<div
							class="text-4xl font-bold"
							class:text-green-500={results.headers.score === 'A+'}
							class:text-yellow-500={results.headers.score === 'B+'}
						>
							{results.headers.score}
						</div>
						<div class="text-gray-400">Security Headers Score</div>
					</div>

					{#if results.headers.issues.length > 0}
						<div class="mb-4">
							<h3 class="text-red-400 flex items-center gap-2 mb-2">
								<AlertTriangle class="w-4 h-4" />
								Issues Found
							</h3>
							<ul class="space-y-1 text-gray-300">
								{#each results.headers.issues as issue}
									<li>• {issue}</li>
								{/each}
							</ul>
						</div>
					{/if}

					<div>
						<h3 class="text-green-400 mb-2">Passed Checks</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.headers.passed as check}
								<li>• {check}</li>
							{/each}
						</ul>
					</div>
				</div>
			{/if}
		</section>

		<!-- Certificate Section -->
		<section>
			<h2 class="text-xl flex items-center gap-2 mb-4">
				<Wrench class="w-5 h-5" />
				SSL/TLS Certificate Analysis
			</h2>

			{#if loading}
				<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
					<div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
					<div class="space-y-2">
						{#each Array(4) as _}
							<div class="h-4 bg-gray-700 rounded w-full"></div>
						{/each}
					</div>
				</div>
			{:else if results}
				<div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
					<div class="flex items-center gap-4 mb-6">
						<div
							class="text-4xl font-bold"
							class:text-green-500={results.certificate.grade === 'A'}
						>
							{results.certificate.grade}
						</div>
						<div class="text-gray-400">Certificate Grade</div>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<div>
							<h3 class="text-gray-400 mb-2">Certificate Details</h3>
							<p class="text-gray-300">Valid until: {results.certificate.validUntil}</p>
							<p class="text-gray-300">Issuer: {results.certificate.issuer}</p>
						</div>

						<div>
							<h3 class="text-gray-400 mb-2">Findings</h3>
							<ul class="space-y-1 text-gray-300">
								{#each results.certificate.findings as finding}
									<li>• {finding}</li>
								{/each}
							</ul>
						</div>
					</div>

					{#if results.certificate.warnings.length > 0}
						<div class="mt-4">
							<h3 class="text-yellow-400 flex items-center gap-2 mb-2">
								<AlertTriangle class="w-4 h-4" />
								Warnings
							</h3>
							<ul class="space-y-1 text-gray-300">
								{#each results.certificate.warnings as warning}
									<li>• {warning}</li>
								{/each}
							</ul>
						</div>
					{/if}
				</div>
			{/if}
		</section>

		<!-- Admin Pages Section -->
		<section>
			<h2 class="text-xl flex items-center gap-2 mb-4">
				<Server class="w-5 h-5" />
				Admin Pages Exposure
			</h2>

			{#if loading}
				<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
					<div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
					<div class="space-y-2">
						{#each Array(3) as _}
							<div class="h-4 bg-gray-700 rounded w-full"></div>
						{/each}
					</div>
				</div>
			{:else if results}
				<div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
					<div class="mb-4">
						<div class="text-lg font-semibold mb-2">
							Risk Level:
							<span class="text-yellow-500 uppercase">{results.adminPages.risk}</span>
						</div>
					</div>

					<div class="mb-4">
						<h3 class="text-red-400 mb-2">Exposed Pages</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.adminPages.exposed as page}
								<li>• {page}</li>
							{/each}
						</ul>
					</div>

					<div>
						<h3 class="text-blue-400 mb-2">Recommendations</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.adminPages.recommendations as rec}
								<li>• {rec}</li>
							{/each}
						</ul>
					</div>
				</div>
			{/if}
		</section>

		<!-- Swagger/API Documentation Section -->
		<section>
			<h2 class="text-xl flex items-center gap-2 mb-4">
				<Book class="w-5 h-5" />
				API Documentation Exposure
			</h2>

			{#if loading}
				<div class="bg-[#202020] rounded-lg p-6 animate-pulse">
					<div class="h-8 bg-gray-700 rounded w-1/4 mb-4"></div>
					<div class="space-y-2">
						{#each Array(3) as _}
							<div class="h-4 bg-gray-700 rounded w-full"></div>
						{/each}
					</div>
				</div>
			{:else if results}
				<div class="bg-[#202020] rounded-lg p-6" in:fade={{ duration: 200 }}>
					<div class="mb-4">
						<div class="text-lg font-semibold mb-2">
							Risk Level:
							<span class="text-red-500 uppercase">{results.swagger.risk}</span>
						</div>
						<div class="text-gray-300">
							Status: {results.swagger.exposed
								? 'API Documentation Exposed'
								: 'No Public API Documentation Found'}
						</div>
					</div>

					{#if results.swagger.endpoints.length > 0}
						<div class="mb-4">
							<h3 class="text-red-400 mb-2">Exposed Endpoints</h3>
							<ul class="space-y-1 text-gray-300">
								{#each results.swagger.endpoints as endpoint}
									<li>• {endpoint}</li>
								{/each}
							</ul>
						</div>
					{/if}

					<div>
						<h3 class="text-blue-400 mb-2">Recommendations</h3>
						<ul class="space-y-1 text-gray-300">
							{#each results.swagger.recommendations as rec}
								<li>• {rec}</li>
							{/each}
						</ul>
					</div>
				</div>
			{/if}
		</section>
	</div>
</div>

<style>
	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.5;
		}
	}

	.animate-pulse {
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}
</style>
