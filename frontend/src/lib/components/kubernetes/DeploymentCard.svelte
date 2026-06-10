<script lang="ts">
	import type { KubeDeployment } from '$lib/api/kubernetes';
	import { scaleDeployment } from '$lib/api/kubernetes';
	import { addToast } from '$lib/stores/ui';

	let {
		deployment,
		onScale,
		context = ''
	}: { deployment: KubeDeployment; onScale: () => void; context?: string } = $props();

	let scaling = $state(false);

	async function updateScale(newVal: number) {
		if (newVal < 0) return;
		scaling = true;
		try {
			const res = await scaleDeployment(deployment.namespace, deployment.name, newVal, context || undefined);
			if (res.success) {
				addToast(res.message || 'Scaling initiated successfully', 'success');
				onScale();
			} else {
				addToast(res.message || 'Failed to scale deployment', 'error');
			}
		} catch (err: any) {
			addToast(err.message || 'Failed to scale deployment', 'error');
		} finally {
			scaling = false;
		}
	}

	const pct = deployment.desired > 0 ? Math.round((deployment.ready / deployment.desired) * 100) : 0;
	const isHealthy = deployment.ready === deployment.desired && deployment.desired > 0;
	const isPartial = deployment.ready > 0 && deployment.ready < deployment.desired;
	const barColor = isHealthy ? 'bg-emerald-500' : isPartial ? 'bg-amber-500' : 'bg-red-500';

	function formatAge(seconds: number): string {
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
		if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
		return `${Math.floor(seconds / 86400)}d`;
	}
</script>

<div class="qp-card p-4 hover:border-white/20 transition-all">
	<div class="flex items-start justify-between mb-3">
		<div>
			<div class="text-white font-medium text-sm">{deployment.name}</div>
			<div class="flex items-center gap-2 mt-1">
				<span class="px-2 py-0.5 rounded text-xs bg-white/10 text-[var(--qp-text-muted)] font-mono">{deployment.namespace}</span>
				<span class="text-xs text-[var(--qp-text-muted)]">{deployment.strategy}</span>
			</div>
		</div>
		<div class="text-right">
			<div class="text-lg font-bold {isHealthy ? 'text-emerald-400' : isPartial ? 'text-amber-400' : 'text-red-400'}">
				{deployment.ready}/{deployment.desired}
			</div>
			<div class="text-xs text-[var(--qp-text-muted)]">replicas</div>
		</div>
	</div>

	<!-- Replica progress bar -->
	<div class="h-1.5 bg-white/10 rounded-full overflow-hidden mb-3">
		<div class="{barColor} h-full rounded-full transition-all duration-500" style="width: {pct}%"></div>
	</div>

	<div class="grid grid-cols-3 gap-2 text-center">
		<div>
			<div class="text-xs text-[var(--qp-text-muted)]">Available</div>
			<div class="text-sm font-medium text-white">{deployment.available}</div>
		</div>
		<div>
			<div class="text-xs text-[var(--qp-text-muted)]">Updated</div>
			<div class="text-sm font-medium text-white">{deployment.updated}</div>
		</div>
		<div>
			<div class="text-xs text-[var(--qp-text-muted)]">Age</div>
			<div class="text-sm font-medium text-white">{formatAge(deployment.age_seconds)}</div>
		</div>
	</div>

	<div class="mt-3 text-xs text-[var(--qp-text-muted)] font-mono truncate mb-4" title={deployment.image}>
		{deployment.image}
	</div>

	<!-- Scale Controls -->
	<div class="mt-4 flex items-center justify-between border-t border-white/5 pt-3">
		<span class="text-xs text-[var(--qp-text-muted)] font-medium">Replicas Scale</span>
		<div class="flex items-center gap-2">
			<button 
				class="w-6 h-6 rounded bg-white/5 border border-white/10 flex items-center justify-center text-white text-xs hover:bg-white/10 transition-colors disabled:opacity-50 font-bold"
				onclick={() => updateScale(deployment.desired - 1)}
				disabled={scaling || deployment.desired <= 0}
			>
				-
			</button>
			<span class="text-xs text-white font-mono min-w-[20px] text-center">{deployment.desired}</span>
			<button 
				class="w-6 h-6 rounded bg-white/5 border border-white/10 flex items-center justify-center text-white text-xs hover:bg-white/10 transition-colors disabled:opacity-50 font-bold"
				onclick={() => updateScale(deployment.desired + 1)}
				disabled={scaling}
			>
				+
			</button>
		</div>
	</div>
</div>
