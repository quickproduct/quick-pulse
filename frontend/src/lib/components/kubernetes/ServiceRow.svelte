<script lang="ts">
	import type { KubeService } from '$lib/api/kubernetes';

	let { service }: { service: KubeService } = $props();

	const typeColor: Record<string, string> = {
		ClusterIP: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
		NodePort: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
		LoadBalancer: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
		ExternalName: 'bg-pink-500/20 text-pink-400 border-pink-500/30'
	};
	const badge = typeColor[service.type] ?? 'bg-white/10 text-white/60 border-white/10';
</script>

<tr class="border-b border-white/5 hover:bg-white/[0.03] transition-colors">
	<td class="px-4 py-3">
		<span class="text-white font-medium text-sm">{service.name}</span>
	</td>
	<td class="px-4 py-3">
		<span class="px-2 py-0.5 rounded text-xs bg-white/10 text-[var(--qp-text-muted)] font-mono">{service.namespace}</span>
	</td>
	<td class="px-4 py-3">
		<span class="px-2 py-0.5 rounded border text-xs font-medium {badge}">{service.type}</span>
	</td>
	<td class="px-4 py-3 text-xs text-[var(--qp-text-muted)] font-mono">{service.cluster_ip}</td>
	<td class="px-4 py-3 text-xs font-mono">
		{#if service.external_ip}
			<span class="text-emerald-400">{service.external_ip}</span>
		{:else}
			<span class="text-[var(--qp-text-muted)]">—</span>
		{/if}
	</td>
	<td class="px-4 py-3">
		<div class="flex flex-wrap gap-1">
			{#each service.ports as p}
				<span class="px-1.5 py-0.5 rounded bg-white/10 text-xs font-mono text-[var(--qp-text-muted)]">
					{p.port}{p.node_port ? `:${p.node_port}` : ''}
					<span class="text-white/40">/{p.protocol}</span>
				</span>
			{/each}
		</div>
	</td>
</tr>
