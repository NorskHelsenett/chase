<script lang="ts">
	interface Props {
		currentResponse?: number;
		avgResponse?: number;
		uptimeDay?: number;
		uptimeMonth?: number;
		certDaysLeft?: number;
		certExpDate?: string;
	}

	let {
		currentResponse = 271,
		avgResponse = 138,
		uptimeDay = 100,
		uptimeMonth = 100,
		certDaysLeft = 258,
		certExpDate = '2024-12-23'
	}: Props = $props();

	let formattedCertDaysLeft = $derived(certExpDate === '0001-01-01T00:00:00Z' ? NaN : certDaysLeft);
	let formattedCertExpDate =
		$derived(certExpDate === '0001-01-01T00:00:00Z' ? 'N/A' : new Date(certExpDate).toLocaleDateString());
</script>

<div class="grid grid-cols-5 gap-8 bg-[#202020] rounded-lg p-4">
	<div class="flex flex-col gap-1">
		<span class="text-gray-400 text-sm">Response</span>
		<span class="text-gray-400 text-xs">(Current)</span>
		<span class="text-white text-xl font-medium">{currentResponse} ms</span>
	</div>

	<div class="flex flex-col gap-1">
		<span class="text-gray-400 text-sm">Avg. Response</span>
		<span class="text-gray-400 text-xs">(24-hour)</span>
		<span class="text-white text-xl font-medium">{avgResponse} ms</span>
	</div>

	<div class="flex flex-col gap-1">
		<span class="text-gray-400 text-sm">Uptime</span>
		<span class="text-gray-400 text-xs">(24-hour)</span>
		<span class="text-white text-xl font-medium">{uptimeDay}%</span>
	</div>

	<div class="flex flex-col gap-1">
		<span class="text-gray-400 text-sm">Uptime</span>
		<span class="text-gray-400 text-xs">(30-day)</span>
		<span class="text-white text-xl font-medium">{uptimeMonth}%</span>
	</div>

	<div class="flex flex-col gap-1">
		<span class="text-gray-400 text-sm">Cert Exp.</span>
		<span class="text-gray-400 text-xs">({formattedCertExpDate})</span>
		<span class="text-white text-xl font-medium">
			{Number.isNaN(formattedCertDaysLeft) ? 'N/A' : `${formattedCertDaysLeft} days`}
		</span>
	</div>
</div>
