<script lang="ts">
	import { onMount } from 'svelte';
	import { getMetricsHistory } from '$lib/api/metrics';
	import { addToast } from '$lib/stores/ui';
	import { liveMetrics } from '$lib/stores/metrics';
	import { wsManager } from '$lib/websocket/manager';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';

	let selectedMetric = $state('cpu');
	let selectedRange = $state('1h');
	let historyData: { time: string; value: number }[] = $state([]);
	let loading = $state(true);

	const metricOptions = [
		{ key: 'cpu', label: 'CPU Usage' },
		{ key: 'memory', label: 'Memory Usage' },
		{ key: 'disk', label: 'Disk Usage' },
		{ key: 'load', label: 'Load Average' },
	];

	const rangeOptions = [
		{ key: '1h', label: '1 Hour' },
		{ key: '24h', label: '24 Hours' },
		{ key: '7d', label: '7 Days' },
	];

	async function loadHistory() {
		loading = true;
		try {
			const result = await getMetricsHistory(selectedMetric, selectedRange);
			historyData = result?.data || [];
		} catch (e: any) {
			historyData = [];
			addToast(e.message || 'Failed to load metrics history', 'error');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		selectedMetric;
		selectedRange;
		loadHistory();
	});

	let currentMetrics = $derived($liveMetrics);

	onMount(() => {
		const unsubscribe = wsManager.onMessage('metrics', (data) => {
			liveMetrics.set(data);
		});
		wsManager.connect('metrics', '/ws/metrics');
		return () => {
			unsubscribe();
			wsManager.disconnect('metrics');
		};
	});

	function buildChartSVG(): string {
		if (!historyData.length) return '';
		const width = 800;
		const height = 200;
		const padding = 40;
		const chartW = width - padding * 2;
		const chartH = height - padding * 2;

		const values = historyData.map((d) => d.value);
		const maxVal = Math.max(...values, 1);
		const minVal = Math.min(...values, 0);

		const points = historyData
			.map((d, i) => {
				const x = padding + (i / (historyData.length - 1 || 1)) * chartW;
				const y = padding + chartH - ((d.value - minVal) / (maxVal - minVal || 1)) * chartH;
				return `${x},${y}`;
			})
			.join(' ');

		const areaPoints = `${padding},${padding + chartH} ${points} ${padding + chartW},${padding + chartH}`;

		return `<svg viewBox="0 0 ${width} ${height}" class="w-full h-auto">
			<defs>
				<linearGradient id="chartGrad" x1="0" y1="0" x2="0" y2="1">
					<stop offset="0%" stop-color="var(--qp-accent)" stop-opacity="0.3"/>
					<stop offset="100%" stop-color="var(--qp-accent)" stop-opacity="0"/>
				</linearGradient>
			</defs>
			<polygon points="${areaPoints}" fill="url(#chartGrad)" />
			<polyline points="${points}" fill="none" stroke="var(--qp-accent)" stroke-width="2" />
		</svg>`;
	}
</script>

<svelte:head>
	<title>Metrics - QuickPulse</title>
</svelte:head>

<PageHeader title="Metrics" subtitle="System performance monitoring" />

<div class="space-y-6">
	<div class="flex items-center gap-3 flex-wrap">
		{#each metricOptions as m}
			<button
				class="qp-btn {selectedMetric === m.key ? 'qp-btn-primary' : 'qp-btn-ghost'} text-xs"
				onclick={() => (selectedMetric = m.key)}
			>
				{m.label}
			</button>
		{/each}
		<div class="flex-1"></div>
		{#each rangeOptions as r}
			<button
				class="qp-btn {selectedRange === r.key ? 'qp-btn-primary' : 'qp-btn-ghost'} text-xs"
				onclick={() => (selectedRange = r.key)}
			>
				{r.label}
			</button>
		{/each}
	</div>

	{#if currentMetrics}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
			<div class="qp-card text-center">
				<div class="text-xs text-[var(--qp-text-muted)] mb-1">CPU</div>
				<div class="text-2xl font-bold text-white">{currentMetrics.cpu_percent?.toFixed(1) ?? 0}%</div>
			</div>
			<div class="qp-card text-center">
				<div class="text-xs text-[var(--qp-text-muted)] mb-1">Memory</div>
				<div class="text-2xl font-bold text-white">{currentMetrics.memory_percent?.toFixed(1) ?? 0}%</div>
			</div>
			<div class="qp-card text-center">
				<div class="text-xs text-[var(--qp-text-muted)] mb-1">Disk</div>
				<div class="text-2xl font-bold text-white">{currentMetrics.disk_percent?.toFixed(1) ?? 0}%</div>
			</div>
			<div class="qp-card text-center">
				<div class="text-xs text-[var(--qp-text-muted)] mb-1">Load</div>
				<div class="text-2xl font-bold text-white">{currentMetrics.load_1m?.toFixed(2) ?? 0}</div>
			</div>
		</div>
	{/if}

	<div class="qp-card min-h-[14rem]">
		<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-4">
			{metricOptions.find((m) => m.key === selectedMetric)?.label || selectedMetric} —
			{rangeOptions.find((r) => r.key === selectedRange)?.label || selectedRange}
		</h3>
		{#if loading}
			<div class="h-48 flex items-center justify-center text-[var(--qp-text-muted)]">
				<svg class="w-5 h-5 animate-spin mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21 12a9 9 0 1 1-6.219-8.56" />
				</svg>
				Loading...
			</div>
		{:else if historyData.length === 0}
			<div class="h-48 flex items-center justify-center text-[var(--qp-text-muted)]">No data available for this range</div>
		{:else}
			{@html buildChartSVG()}
		{/if}
	</div>
</div>
