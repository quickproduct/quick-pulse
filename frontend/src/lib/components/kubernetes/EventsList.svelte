<script lang="ts">
	import type { KubeEvent } from '$lib/api/kubernetes';

	let { events }: { events: KubeEvent[] } = $props();

	function formatAge(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
		if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
		return `${Math.floor(seconds / 86400)}d`;
	}
</script>

<div class="qp-card overflow-hidden">
	<div class="p-4 border-b border-white/5 bg-white/[0.02] flex items-center justify-between">
		<h3 class="text-white font-semibold text-sm">Cluster Events</h3>
		<span class="text-xs text-[var(--qp-text-muted)]">{events.length} events logged</span>
	</div>
	<div class="overflow-y-auto max-h-[400px]">
		<table class="w-full text-left border-collapse">
			<thead class="sticky top-0 bg-[#0B0E14] shadow-sm">
				<tr class="text-[var(--qp-text-muted)] text-[10px] uppercase tracking-widest font-bold border-b border-white/5">
					<th class="px-4 py-2">Type</th>
					<th class="px-4 py-2">Reason</th>
					<th class="px-4 py-2">Object</th>
					<th class="px-4 py-2">Message</th>
					<th class="px-4 py-2 text-right">Age</th>
				</tr>
			</thead>
			<tbody>
				{#each events as event}
					<tr class="border-b border-white/5 hover:bg-white/[0.03] transition-colors text-xs">
						<td class="px-4 py-3">
							<span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase {event.type === 'Warning' ? 'bg-red-500/20 text-red-400 border border-red-500/30' : 'bg-white/10 text-[var(--qp-text-muted)]'}">
								{event.type}
							</span>
						</td>
						<td class="px-4 py-3 text-white font-medium">{event.reason}</td>
						<td class="px-4 py-3 text-[var(--qp-text-muted)] font-mono">{event.object}</td>
						<td class="px-4 py-3 text-[var(--qp-text-muted)] leading-relaxed max-w-xs">{event.message}</td>
						<td class="px-4 py-3 text-right text-[var(--qp-text-muted)] whitespace-nowrap">{formatAge(event.age_seconds)}</td>
					</tr>
				{/each}
				{#if events.length === 0}
					<tr>
						<td colspan="5" class="px-4 py-8 text-center text-[var(--qp-text-muted)]">No recent events</td>
					</tr>
				{/if}
			</tbody>
		</table>
	</div>
</div>
