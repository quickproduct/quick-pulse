<script lang="ts">
	let { title = '', value = '', unit = '', icon = '', trend = '' } = $props();

	const iconPaths: Record<string, string> = {
		cpu: 'M22 12h-4l-3 9L9 3l-3 9H2',
		memory: 'M6 19v-8M10 19V5M14 19v-4M18 19v-10',
		disk: 'M22 12H2M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z',
		network: 'M12 2a10 10 0 1 0 10 10M12 2v10l8.5-5',
		load: 'M18 20V10M12 20V4M6 20v-6',
		containers: 'M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z',
	};

	function getColor(icon: string): string {
		const colors: Record<string, string> = {
			cpu: 'text-blue-400',
			memory: 'text-purple-400',
			disk: 'text-amber-400',
			network: 'text-green-400',
			load: 'text-rose-400',
			containers: 'text-indigo-400',
		};
		return colors[icon] || 'text-[var(--qp-accent)]';
	}
</script>

<div class="qp-card flex items-center gap-4">
	<div class="flex items-center justify-center w-10 h-10 rounded-lg bg-[var(--qp-surface-2)] {getColor(icon)}">
		<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
			<path d={iconPaths[icon] || iconPaths.cpu} />
		</svg>
	</div>
	<div class="flex-1 min-w-0">
		<p class="text-xs text-[var(--qp-text-muted)] uppercase tracking-wide">{title}</p>
		<div class="flex items-baseline gap-1">
			<span class="text-2xl font-bold text-white">{value}</span>
			{#if unit}
				<span class="text-xs text-[var(--qp-text-muted)]">{unit}</span>
			{/if}
		</div>
	</div>
</div>
