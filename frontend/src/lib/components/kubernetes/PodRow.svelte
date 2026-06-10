<script lang="ts">
	import type { KubePod } from '$lib/api/kubernetes';
	import { deletePod } from '$lib/api/kubernetes';
	import { addToast } from '$lib/stores/ui';

	let {
		pod,
		onDelete,
		context = ''
	}: { pod: KubePod; onDelete: () => void; context?: string } = $props();

	let deleting = $state(false);
	let confirming = $state(false);
	let showInfo = $state(false);

	async function handleDelete() {
		deleting = true;
		try {
			const res = await deletePod(pod.namespace, pod.name, context || undefined);
			if (res.success) {
				addToast(res.message || 'Pod deleted successfully', 'success');
				onDelete();
			} else {
				addToast(res.message || 'Failed to delete pod', 'error');
			}
		} catch (err: any) {
			addToast(err.message || 'Failed to delete pod', 'error');
		} finally {
			deleting = false;
			confirming = false;
		}
	}

	function formatAge(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
		if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
		return `${Math.floor(seconds / 86400)}d`;
	}

	const statusColor: Record<string, string> = {
		Running: 'text-emerald-400',
		Pending: 'text-amber-400',
		Failed: 'text-red-400',
		Succeeded: 'text-blue-400',
		Unknown: 'text-gray-400',
		CrashLoopBackOff: 'text-red-500'
	};

	const statusDot: Record<string, string> = {
		Running: 'bg-emerald-400',
		Pending: 'bg-amber-400 animate-pulse',
		Failed: 'bg-red-400',
		Succeeded: 'bg-blue-400',
		Unknown: 'bg-gray-400',
		CrashLoopBackOff: 'bg-red-500 animate-pulse'
	};

	const color = statusColor[pod.status] ?? 'text-gray-400';
	const dot = statusDot[pod.status] ?? 'bg-gray-400';
</script>

<tr class="border-b border-white/5 hover:bg-white/[0.03] transition-colors">
	<td class="px-4 py-3">
		<div class="flex items-center gap-2">
			<span class="w-2 h-2 rounded-full {dot} flex-shrink-0"></span>
			<span class="text-white font-mono text-xs truncate max-w-[180px]" title={pod.name}>{pod.name}</span>
		</div>
	</td>
	<td class="px-4 py-3">
		<span class="px-2 py-0.5 rounded text-xs bg-white/10 text-[var(--qp-text-muted)] font-mono">{pod.namespace}</span>
	</td>
	<td class="px-4 py-3">
		<span class="text-xs font-medium {color}">{pod.status}</span>
	</td>
	<td class="px-4 py-3 text-xs text-[var(--qp-text-muted)] font-mono">{pod.ready}</td>
	<td class="px-4 py-3 text-xs text-center">
		<span class={pod.restarts > 0 ? (pod.restarts >= 5 ? 'text-red-400 font-bold' : 'text-amber-400') : 'text-[var(--qp-text-muted)]'}>
			{pod.restarts}
		</span>
	</td>
	<td class="px-4 py-3 text-xs text-[var(--qp-text-muted)] truncate max-w-[120px]" title={pod.node}>{pod.node || '—'}</td>
	<td class="px-4 py-3 text-xs text-[var(--qp-text-muted)]">{formatAge(pod.age_seconds)}</td>
	<td class="px-4 py-3 text-xs text-right space-x-2">
		<button class="text-[var(--qp-accent)] hover:underline mr-1" onclick={() => showInfo = true}>Info</button>
		<a href="/kubernetes/pods/{pod.namespace}/{pod.name}/logs{context ? `?context=${encodeURIComponent(context)}` : ''}" class="text-[var(--qp-accent)] hover:underline mr-1">Logs</a>
		{#if confirming}
			<button class="text-red-400 font-bold hover:underline" onclick={handleDelete} disabled={deleting}>
				{deleting ? '...' : 'Confirm'}
			</button>
			<button class="text-[var(--qp-text-muted)] hover:underline" onclick={() => confirming = false} disabled={deleting}>
				Cancel
			</button>
		{:else}
			<button class="text-red-400 hover:underline" onclick={() => confirming = true}>
				Delete
			</button>
		{/if}
	</td>
</tr>

{#if showInfo}
	<!-- Modal overlay -->
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in" onclick={() => showInfo = false} role="presentation">
		<div class="w-full max-w-2xl bg-[var(--qp-surface)] border border-[var(--qp-border)] rounded-xl shadow-2xl p-6 text-left space-y-6 m-4" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-labelledby="modal-title">
			<!-- Header -->
			<div class="flex items-start justify-between border-b border-white/[0.05] pb-4">
				<div>
					<h3 id="modal-title" class="text-base font-bold text-white font-mono break-all">{pod.name}</h3>
					<p class="text-xs text-[var(--qp-text-muted)] mt-1 font-mono">Namespace: {pod.namespace}</p>
				</div>
				<button class="text-[var(--qp-text-muted)] hover:text-white transition-colors p-1" onclick={() => showInfo = false} aria-label="Close modal">
					<svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Grid specs -->
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
				<div class="space-y-1">
					<span class="text-[10px] text-[var(--qp-text-muted)] block uppercase tracking-wider">Status</span>
					<span class="font-semibold flex items-center gap-1.5 {color}">
						<span class="w-2 h-2 rounded-full {dot}"></span>
						{pod.status}
					</span>
				</div>
				<div class="space-y-1">
					<span class="text-[10px] text-[var(--qp-text-muted)] block uppercase tracking-wider">Scheduling Node</span>
					<span class="font-mono text-white text-xs break-all">{pod.node || '—'}</span>
				</div>
				<div class="space-y-1">
					<span class="text-[10px] text-[var(--qp-text-muted)] block uppercase tracking-wider">Containers Status</span>
					<span class="font-mono text-white font-medium">{pod.ready} Ready</span>
				</div>
				<div class="space-y-1">
					<span class="text-[10px] text-[var(--qp-text-muted)] block uppercase tracking-wider">Restart Count</span>
					<span class="font-mono font-medium text-white">{pod.restarts} restarts</span>
				</div>
				<div class="space-y-1">
					<span class="text-[10px] text-[var(--qp-text-muted)] block uppercase tracking-wider">CPU Request</span>
					<span class="font-mono text-white font-medium">{pod.cpu || '—'}</span>
				</div>
				<div class="space-y-1">
					<span class="text-[10px] text-[var(--qp-text-muted)] block uppercase tracking-wider">Memory Request</span>
					<span class="font-mono text-white font-medium">{pod.memory || '—'}</span>
				</div>
				<div class="space-y-1 sm:col-span-2">
					<span class="text-[10px] text-[var(--qp-text-muted)] block uppercase tracking-wider">Container Image</span>
					<span class="font-mono text-indigo-300 text-xs break-all">{pod.image || '—'}</span>
				</div>
			</div>

			<!-- Pod Conditions -->
			{#if (pod as any).conditions && (pod as any).conditions.length > 0}
				<div class="border-t border-white/[0.05] pt-4">
					<h4 class="text-[10px] font-bold text-white uppercase tracking-wider mb-3">Lifecycle Conditions</h4>
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs font-mono">
						{#each (pod as any).conditions as cond}
							<div class="flex items-center justify-between p-2 bg-white/[0.02] border border-white/[0.05] rounded">
								<span class="text-[var(--qp-text-muted)]">{cond.type}</span>
								<span class="font-bold {cond.status === 'True' ? 'text-emerald-400' : 'text-amber-400'}">
									{cond.status === 'True' ? '✓ True' : '✗ False'}
								</span>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Footer details -->
			<div class="border-t border-white/[0.05] pt-4 flex justify-between items-center text-xs">
				<span class="text-[var(--qp-text-muted)]">Age: {formatAge(pod.age_seconds)}</span>
				<a href="/kubernetes/pods/{pod.namespace}/{pod.name}/logs{context ? `?context=${encodeURIComponent(context)}` : ''}" class="qp-btn qp-btn-ghost text-xs" onclick={() => showInfo = false}>
					Stream Full Logs
				</a>
			</div>
		</div>
	</div>
{/if}
