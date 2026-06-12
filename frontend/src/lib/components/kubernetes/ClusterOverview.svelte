<script lang="ts">
	import type { ClusterOverview } from '$lib/api/kubernetes';

	let { overview }: { overview: ClusterOverview } = $props();

	const podPct = overview.pods_total > 0 ? Math.round((overview.pods_running / overview.pods_total) * 100) : 0;
	const nodePct = overview.nodes > 0 ? Math.round((overview.nodes_ready / overview.nodes) * 100) : 0;
</script>

<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
	<div class="qp-card p-4">
		<div class="text-[var(--qp-text-muted)] text-xs mb-1 uppercase tracking-wider font-semibold">Nodes</div>
		<div class="flex items-end justify-between">
			<div class="text-2xl font-bold text-white">{overview.nodes_ready}<span class="text-sm text-[var(--qp-text-muted)] font-normal ml-1">/ {overview.nodes}</span></div>
			<div class="text-xs {overview.nodes_ready === overview.nodes ? 'text-emerald-400' : 'text-amber-400'}">
				{nodePct}% Ready
			</div>
		</div>
		<div class="h-1 bg-white/5 rounded-full mt-2 overflow-hidden">
			<div class="h-full {overview.nodes_ready === overview.nodes ? 'bg-emerald-500' : 'bg-amber-500'}" style="width: {nodePct}%"></div>
		</div>
	</div>

	<div class="qp-card p-4">
		<div class="text-[var(--qp-text-muted)] text-xs mb-1 uppercase tracking-wider font-semibold">Pods</div>
		<div class="flex items-end justify-between">
			<div class="text-2xl font-bold text-white">{overview.pods_running}<span class="text-sm text-[var(--qp-text-muted)] font-normal ml-1">/ {overview.pods_total}</span></div>
			<div class="text-xs {podPct > 90 ? 'text-emerald-400' : 'text-amber-400'}">
				{podPct}% Running
			</div>
		</div>
		<div class="h-1 bg-white/5 rounded-full mt-2 overflow-hidden">
			<div class="h-full {podPct > 90 ? 'bg-emerald-500' : 'bg-amber-500'}" style="width: {podPct}%"></div>
		</div>
	</div>

	<div class="qp-card p-4">
		<div class="text-[var(--qp-text-muted)] text-xs mb-1 uppercase tracking-wider font-semibold">Namespaces</div>
		<div class="text-2xl font-bold text-white">{overview.namespaces}</div>
		<div class="text-xs text-[var(--qp-text-muted)] mt-1">Active in cluster</div>
	</div>

	<div class="qp-card p-4">
		<div class="text-[var(--qp-text-muted)] text-xs mb-1 uppercase tracking-wider font-semibold">Cluster</div>
		<div class="flex items-center gap-2">
			<div class="text-xl font-bold {overview.source === 'live' ? 'text-emerald-400' : 'text-amber-400'}">
				{overview.source === 'live' ? 'Live Cluster' : 'Not Connected'}
			</div>
			<div class="w-2 h-2 rounded-full {overview.source === 'live' ? 'bg-emerald-400' : 'bg-amber-400 animate-pulse'}"></div>
		</div>
		{#if overview.source === 'live'}
			<div class="text-xs text-[var(--qp-text-muted)] mt-1">Connected to K8s API</div>
		{:else}
			<div class="text-xs text-[var(--qp-text-muted)] mt-1" title={overview.reason}>
				{overview.reason || 'No cluster detected'}
			</div>
			<div class="text-[10px] text-[var(--qp-text-muted)] mt-1 opacity-70">
				Set <code>KUBECONFIG</code> or mount <code>~/.kube/config</code> to connect.
			</div>
		{/if}
	</div>
</div>
