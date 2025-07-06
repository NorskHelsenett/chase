<script>
	import { getRiskColor } from '$lib/utils';
	import {
		Wrench,
		AlertTriangle,
		Shield,
		Calendar,
		Building2,
		Award,
		Key,
		Hash
	} from 'lucide-svelte';
	import { fade } from 'svelte/transition';

	export let loading = true;
	export let results = {};
</script>

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
			<!-- Grade Section -->
			<div class="flex items-center gap-4 mb-6">
				<div class="text-4xl font-bold {getRiskColor(results.certificate.grade)}">
					{results.certificate.grade}
				</div>
				<div class="text-gray-400">Certificate Grade</div>
			</div>

			<div class="grid grid-cols-2 gap-6">
				<!-- Certificate Details -->
				<div>
					<h3 class="text-gray-400 flex items-center gap-2 mb-4">
						<Shield class="w-4 h-4" />
						Certificate Details
					</h3>

					<div class="space-y-4">
						<!-- Validity Period -->
						<div>
							<div class="flex items-center gap-2 mb-2">
								<Calendar class="w-4 h-4 text-gray-400" />
								<h4 class="text-gray-400">Validity Period</h4>
							</div>
							<div class="grid grid-cols-[auto,1fr] gap-x-3 text-sm">
								<span class="text-gray-400">Valid from:</span>
								<span class="text-gray-300"
									>{new Date(results.certificate.validFrom).toLocaleString(undefined, {
										dateStyle: 'medium',
										timeStyle: 'short'
									})}</span
								>
								<span class="text-gray-400">Valid until:</span>
								<span class="text-gray-300"
									>{new Date(results.certificate.validUntil).toLocaleString(undefined, {
										dateStyle: 'medium',
										timeStyle: 'short'
									})}</span
								>
							</div>
						</div>

						<!-- Organization Info -->
						<div>
							<div class="flex items-center gap-2 mb-2">
								<Building2 class="w-4 h-4 text-gray-400" />
								<h4 class="text-gray-400">Organization Info</h4>
							</div>
							<div class="grid grid-cols-[auto,1fr] gap-x-3 text-sm">
								<span class="text-gray-400">Organization:</span>
								<span class="text-gray-300">{results.certificate.organization || 'Unknown'}</span>
								<span class="text-gray-400">Issuer:</span>
								<span class="text-gray-300">{results.certificate.issuer || 'Unknown'}</span>
							</div>
						</div>

						<!-- Technical Details -->
						<div>
							<div class="flex items-center gap-2 mb-2">
								<Key class="w-4 h-4 text-gray-400" />
								<h4 class="text-gray-400">Technical Details</h4>
							</div>
							<div class="grid grid-cols-[auto,1fr] gap-x-3 text-sm">
								<span class="text-gray-400">Key Type:</span>
								<span class="text-gray-300"
									>{results.certificate.publicKeyType} ({results.certificate.publicKeyBits} bits)</span
								>
								<span class="text-gray-400">Signature:</span>
								<span class="text-gray-300">{results.certificate.signatureAlg || 'Unknown'}</span>
								<span class="text-gray-400">Serial:</span>
								<span class="text-gray-300 font-mono text-xs"
									>{results.certificate.serialNumber || 'Unknown'}</span
								>
							</div>
						</div>

						<!-- Subject DNS Names -->
						{#if results.certificate.subjectDNS && results.certificate.subjectDNS.length > 0}
							<div>
								<div class="flex items-center gap-2 mb-2">
									<Hash class="w-4 h-4 text-gray-400" />
									<h4 class="text-gray-400">Protected Domains</h4>
								</div>
								<ul class="text-sm text-gray-300">
									{#each results.certificate.subjectDNS as dns}
										<li class="font-mono">• {dns}</li>
									{/each}
								</ul>
							</div>
						{/if}
					</div>
				</div>

				<!-- Findings and Warnings -->
				<div class="space-y-6">
					<!-- Findings -->
					{#if results.certificate.findings.length > 0}
						<div>
							<h3 class="text-gray-400 flex items-center gap-2 mb-2">
								<Award class="w-4 h-4" />
								Findings
							</h3>
							<ul class="space-y-2">
								{#each results.certificate.findings as finding}
									<li class="text-gray-300 text-sm">
										<div class="font-medium">{finding.description}</div>
										<div class="text-gray-400 text-xs mt-1">{finding.evidence}</div>
									</li>
								{/each}
							</ul>
						</div>
					{/if}

					<!-- Warnings -->
					{#if results.certificate.warnings.length > 0}
						<div>
							<h3 class="text-yellow-400 flex items-center gap-2 mb-2">
								<AlertTriangle class="w-4 h-4" />
								Warnings
							</h3>
							<ul class="space-y-2">
								{#each results.certificate.warnings as warning}
									<li class="text-gray-300 text-sm">
										<div class="font-medium">{warning.description}</div>
										<div class="text-gray-400 text-xs mt-1">{warning.evidence}</div>
									</li>
								{/each}
							</ul>
						</div>
					{/if}

					<!-- TLS Versions -->
					{#if results.certificate.tlsVersions.length > 0}
						<div>
							<h3 class="text-gray-400 flex items-center gap-2 mb-2">
								<Shield class="w-4 h-4" />
								TLS Versions
							</h3>
							<ul class="space-y-1 text-sm">
								{#each results.certificate.tlsVersions as version}
									<li class="text-gray-300">• {version}</li>
								{/each}
							</ul>
						</div>
					{/if}
				</div>
			</div>
		</div>
	{/if}
</section>
