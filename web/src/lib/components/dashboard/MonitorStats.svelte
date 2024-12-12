<script lang="ts">
  export let stats = {
    up: 5,
    down: 1,
    headerScoreAvg: 'A',
    certScoreAvg: 'A',
    criticalRisks: 1,
    highRisks: 1
  };
</script>

<div class="bg-[#202020] rounded-lg p-4 mb-4">
  <div class="grid grid-cols-6 gap-8">
    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Online</div>
      <div class="text-green-400 text-2xl font-semibold">{stats.up}</div>
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Offline</div>
      <div class="text-red-400 text-2xl font-semibold">{stats.down}</div>
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Avg Header</div>
      <div class="text-blue-400 text-2xl font-semibold">{stats.headerScoreAvg}</div>
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Avg Cert</div>
      <div class="text-blue-400 text-2xl font-semibold">{stats.certScoreAvg}</div>
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">Critical Risks</div>
      <div class="text-red-400 text-2xl font-semibold">{stats.criticalRisks}</div>
    </div>

    <div class="text-center">
      <div class="text-gray-400 text-lg mb-2">High Risks</div>
      <div class="text-orange-400 text-2xl font-semibold">{stats.highRisks}</div>
    </div>
  </div>
</div>