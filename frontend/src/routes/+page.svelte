<script lang="ts">
	import { onMount } from 'svelte';
	import { wsManager } from '$lib/websocket/manager';
	import { liveMetrics } from '$lib/stores/metrics';
	import { getDashboard } from '$lib/api/alerts';
	import { addToast } from '$lib/stores/ui';
	import StatusBadge from '$lib/components/shared/StatusBadge.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';
	import LoadingSkeleton from '$lib/components/shared/LoadingSkeleton.svelte';

	let dashboard: any = $state(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	function formatBytes(bytes: number): string {
		if (!bytes || bytes <= 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
	}

	function formatUptime(seconds: number): string {
		if (!seconds || seconds <= 0) return '0m';
		const days = Math.floor(seconds / 86400);
		const hours = Math.floor((seconds % 86400) / 3600);
		if (days > 0) return `${days}d ${hours}h`;
		const mins = Math.floor((seconds % 3600) / 60);
		return `${hours}h ${mins}m`;
	}

	onMount(() => {
		async function init() {
			try {
				dashboard = await getDashboard();
			} catch (e: any) {
				error = e.message || 'Failed to load dashboard';
				addToast(error ?? 'Failed to load dashboard', 'error');
			} finally {
				loading = false;
			}
		}
		init();

		const unsubscribe = wsManager.onMessage('metrics', (data) => {
			liveMetrics.set(data);
		});
		wsManager.connect('metrics', '/ws/metrics');

		return () => {
			unsubscribe();
			wsManager.disconnect('metrics');
		};
	});

	let metrics = $derived($liveMetrics || dashboard?.metrics);

	// Circle circumference calculations (Radius = 50, C = 314.16)
	const CIRCUMFERENCE = 314.16;
	let cpuOffset = $derived(CIRCUMFERENCE - ((metrics?.cpu_percent ?? 0) / 100) * CIRCUMFERENCE);
	let memOffset = $derived(CIRCUMFERENCE - ((metrics?.memory_percent ?? 0) / 100) * CIRCUMFERENCE);
	let diskOffset = $derived(CIRCUMFERENCE - ((metrics?.disk_percent ?? 0) / 100) * CIRCUMFERENCE);
</script>

<svelte:head>
	<title>Dashboard - QuickPulse</title>
</svelte:head>

<div class="max-w-7xl mx-auto space-y-6">
	<PageHeader title="Dashboard" subtitle="Real-time system health and resource consumption overview." />

	{#if loading}
		<LoadingSkeleton rows={4} />
	{:else if error}
		<div class="qp-card border-red-500/30 bg-red-500/5 text-red-400 text-sm mb-4">
			Failed to load dashboard: {error}
		</div>
	{:else}
		<!-- Main Resource Usage Rings -->
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
			<!-- CPU Ring Card -->
			<div class="qp-card flex flex-col items-center justify-center p-8 relative overflow-hidden group">
				<div class="absolute -right-16 -top-16 w-32 h-32 bg-indigo-500/10 rounded-full blur-2xl group-hover:bg-indigo-500/15 transition-all"></div>
				<h3 class="text-sm font-semibold uppercase tracking-wider text-[var(--qp-text-muted)] mb-6">Processor Load</h3>
				<div class="relative w-36 h-36 flex items-center justify-center">
					<svg class="w-full h-full transform -rotate-90">
						<circle cx="72" cy="72" r="50" stroke="rgba(255,255,255,0.03)" stroke-width="8" fill="none" />
						<circle cx="72" cy="72" r="50" stroke="url(#cpuGrad)" stroke-width="10" fill="none"
							stroke-dasharray={CIRCUMFERENCE} stroke-dashoffset={cpuOffset} stroke-linecap="round"
							class="transition-all duration-700 ease-out" />
						<defs>
							<linearGradient id="cpuGrad" x1="0%" y1="0%" x2="100%" y2="100%">
								<stop offset="0%" stop-color="#818cf8" />
								<stop offset="100%" stop-color="#4f46e5" />
							</linearGradient>
						</defs>
					</svg>
					<div class="absolute inset-0 flex flex-col items-center justify-center">
						<div class="flex items-baseline justify-center">
							<span class="text-2xl font-extrabold text-white font-mono tracking-tight">{metrics?.cpu_percent?.toFixed(1) ?? '0.0'}</span>
							<span class="text-xs font-semibold text-[var(--qp-text-muted)] ml-0.5">%</span>
						</div>
						<span class="text-[10px] font-bold text-[var(--qp-text-muted)] uppercase tracking-widest mt-0.5">CPU</span>
					</div>
				</div>
				<div class="mt-6 text-center">
					<span class="text-xs text-[var(--qp-text-muted)]">Load Average:</span>
					<span class="text-xs font-mono font-bold text-white ml-1">{metrics?.load_1m?.toFixed(2) ?? '0.00'}</span>
				</div>
			</div>

			<!-- Memory Ring Card -->
			<div class="qp-card flex flex-col items-center justify-center p-8 relative overflow-hidden group">
				<div class="absolute -right-16 -top-16 w-32 h-32 bg-cyan-500/10 rounded-full blur-2xl group-hover:bg-cyan-500/15 transition-all"></div>
				<h3 class="text-sm font-semibold uppercase tracking-wider text-[var(--qp-text-muted)] mb-6">Memory Utilisation</h3>
				<div class="relative w-36 h-36 flex items-center justify-center">
					<svg class="w-full h-full transform -rotate-90">
						<circle cx="72" cy="72" r="50" stroke="rgba(255,255,255,0.03)" stroke-width="8" fill="none" />
						<circle cx="72" cy="72" r="50" stroke="url(#memGrad)" stroke-width="10" fill="none"
							stroke-dasharray={CIRCUMFERENCE} stroke-dashoffset={memOffset} stroke-linecap="round"
							class="transition-all duration-700 ease-out" />
						<defs>
							<linearGradient id="memGrad" x1="0%" y1="0%" x2="100%" y2="100%">
								<stop offset="0%" stop-color="#22d3ee" />
								<stop offset="100%" stop-color="#0891b2" />
							</linearGradient>
						</defs>
					</svg>
					<div class="absolute inset-0 flex flex-col items-center justify-center">
						<div class="flex items-baseline justify-center">
							<span class="text-2xl font-extrabold text-white font-mono tracking-tight">{metrics?.memory_percent?.toFixed(1) ?? '0.0'}</span>
							<span class="text-xs font-semibold text-[var(--qp-text-muted)] ml-0.5">%</span>
						</div>
						<span class="text-[10px] font-bold text-[var(--qp-text-muted)] uppercase tracking-widest mt-0.5">RAM</span>
					</div>
				</div>
				<div class="mt-6 text-center">
					<span class="text-xs text-[var(--qp-text-muted)]">Active Processes:</span>
					<span class="text-xs font-mono font-bold text-white ml-1">{metrics?.process_count ?? 0}</span>
				</div>
			</div>

			<!-- Disk Ring Card -->
			<div class="qp-card flex flex-col items-center justify-center p-8 relative overflow-hidden group">
				<div class="absolute -right-16 -top-16 w-32 h-32 bg-amber-500/10 rounded-full blur-2xl group-hover:bg-amber-500/15 transition-all"></div>
				<h3 class="text-sm font-semibold uppercase tracking-wider text-[var(--qp-text-muted)] mb-6">Storage Volume</h3>
				<div class="relative w-36 h-36 flex items-center justify-center">
					<svg class="w-full h-full transform -rotate-90">
						<circle cx="72" cy="72" r="50" stroke="rgba(255,255,255,0.03)" stroke-width="8" fill="none" />
						<circle cx="72" cy="72" r="50" stroke="url(#diskGrad)" stroke-width="10" fill="none"
							stroke-dasharray={CIRCUMFERENCE} stroke-dashoffset={diskOffset} stroke-linecap="round"
							class="transition-all duration-700 ease-out" />
						<defs>
							<linearGradient id="diskGrad" x1="0%" y1="0%" x2="100%" y2="100%">
								<stop offset="0%" stop-color="#f59e0b" />
								<stop offset="100%" stop-color="#d97706" />
							</linearGradient>
						</defs>
					</svg>
					<div class="absolute inset-0 flex flex-col items-center justify-center">
						<div class="flex items-baseline justify-center">
							<span class="text-2xl font-extrabold text-white font-mono tracking-tight">{metrics?.disk_percent?.toFixed(1) ?? '0.0'}</span>
							<span class="text-xs font-semibold text-[var(--qp-text-muted)] ml-0.5">%</span>
						</div>
						<span class="text-[10px] font-bold text-[var(--qp-text-muted)] uppercase tracking-widest mt-0.5">DISK</span>
					</div>
				</div>
				<div class="mt-6 text-center">
					<span class="text-xs text-[var(--qp-text-muted)]">System Uptime:</span>
					<span class="text-xs font-mono font-bold text-white ml-1">{formatUptime(metrics?.uptime_seconds ?? 0)}</span>
				</div>
			</div>
		</div>

		<!-- Secondary metrics & details grid -->
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
			<!-- Left side network stats -->
			<div class="qp-card p-6 flex flex-col justify-between">
				<h3 class="text-sm font-bold uppercase tracking-wider text-[var(--qp-text-muted)] mb-4">Network Activity</h3>
				<div class="space-y-4">
					<div class="flex items-center justify-between p-3 bg-white/[0.02] border border-white/[0.05] rounded-lg">
						<div class="flex items-center gap-3">
							<div class="w-8 h-8 rounded bg-emerald-500/10 flex items-center justify-center text-emerald-400">
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
								</svg>
							</div>
							<div>
								<div class="text-xs text-[var(--qp-text-muted)]">Incoming Traffic</div>
								<div class="text-base font-bold text-white font-mono">{formatBytes(metrics?.net_bytes_recv ?? 0)}</div>
							</div>
						</div>
					</div>
					<div class="flex items-center justify-between p-3 bg-white/[0.02] border border-white/[0.05] rounded-lg">
						<div class="flex items-center gap-3">
							<div class="w-8 h-8 rounded bg-blue-500/10 flex items-center justify-center text-blue-400">
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M5 10l7-7m0 0l7 7m-7-7v18" />
								</svg>
							</div>
							<div>
								<div class="text-xs text-[var(--qp-text-muted)]">Outgoing Traffic</div>
								<div class="text-base font-bold text-white font-mono">{formatBytes(metrics?.net_bytes_sent ?? 0)}</div>
							</div>
						</div>
					</div>
				</div>
				<div class="border-t border-white/[0.05] pt-4 mt-4 flex justify-between items-center text-xs text-[var(--qp-text-muted)]">
					<span>Real-time updates every 10s</span>
					<span class="flex items-center gap-1.5">
						<span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span> Connected
					</span>
				</div>
			</div>

			<!-- Containers Status -->
			<div class="qp-card p-6 flex flex-col justify-between">
				<h3 class="text-sm font-bold uppercase tracking-wider text-[var(--qp-text-muted)] mb-4">Containers Status</h3>
				{#if dashboard?.containers}
					<div class="grid grid-cols-3 gap-2 text-center my-auto">
						<div class="p-3 bg-white/[0.02] border border-white/[0.05] rounded-lg">
							<div class="text-2xl font-bold text-white font-mono">{dashboard.containers.total ?? 0}</div>
							<div class="text-[10px] text-[var(--qp-text-muted)] uppercase mt-1">Total</div>
						</div>
						<div class="p-3 bg-emerald-500/5 border border-emerald-500/10 rounded-lg">
							<div class="text-2xl font-bold text-emerald-400 font-mono">{dashboard.containers.running ?? 0}</div>
							<div class="text-[10px] text-[var(--qp-text-muted)] uppercase mt-1">Running</div>
						</div>
						<div class="p-3 bg-red-500/5 border border-red-500/10 rounded-lg">
							<div class="text-2xl font-bold text-red-400 font-mono">{dashboard.containers.stopped ?? 0}</div>
							<div class="text-[10px] text-[var(--qp-text-muted)] uppercase mt-1">Stopped</div>
						</div>
					</div>
				{:else}
					<EmptyState message="No container data available" />
				{/if}
				<div class="border-t border-white/[0.05] pt-4 mt-4 text-xs text-right">
					<a href="/containers" class="text-[var(--qp-accent)] hover:underline">Manage Containers &rarr;</a>
				</div>
			</div>

			<!-- Stack Health -->
			<div class="qp-card p-6 flex flex-col justify-between">
				<h3 class="text-sm font-bold uppercase tracking-wider text-[var(--qp-text-muted)] mb-4">Docker Compose Stacks</h3>
				{#if dashboard?.stack_summary && dashboard.stack_summary.total > 0}
					<div class="grid grid-cols-4 gap-2 text-center my-auto">
						<div class="p-2.5 bg-white/[0.02] border border-white/[0.05] rounded-lg">
							<div class="text-xl font-bold text-white font-mono">{dashboard.stack_summary.total ?? 0}</div>
							<div class="text-[9px] text-[var(--qp-text-muted)] uppercase mt-1">Total</div>
						</div>
						<div class="p-2.5 bg-emerald-500/5 border border-emerald-500/10 rounded-lg">
							<div class="text-xl font-bold text-emerald-400 font-mono">{dashboard.stack_summary.running ?? 0}</div>
							<div class="text-[9px] text-[var(--qp-text-muted)] uppercase mt-1">Healthy</div>
						</div>
						<div class="p-2.5 bg-amber-500/5 border border-amber-500/10 rounded-lg">
							<div class="text-xl font-bold text-amber-400 font-mono">{dashboard.stack_summary.partial ?? 0}</div>
							<div class="text-[9px] text-[var(--qp-text-muted)] uppercase mt-1">Partial</div>
						</div>
						<div class="p-2.5 bg-red-500/5 border border-red-500/10 rounded-lg">
							<div class="text-xl font-bold text-red-400 font-mono">{dashboard.stack_summary.stopped ?? 0}</div>
							<div class="text-[9px] text-[var(--qp-text-muted)] uppercase mt-1">Stopped</div>
						</div>
					</div>
				{:else}
					<div class="text-center py-6 text-xs text-[var(--qp-text-muted)]">No active Docker Compose stacks detected.</div>
				{/if}
				<div class="border-t border-white/[0.05] pt-4 mt-4 text-xs text-right">
					<a href="/stacks" class="text-[var(--qp-accent)] hover:underline">Manage Stacks &rarr;</a>
				</div>
			</div>
		</div>

		<!-- Activity Log / Recent Events timeline -->
		<div class="qp-card p-6">
			<div class="flex items-center justify-between mb-6">
				<h3 class="text-sm font-bold uppercase tracking-wider text-[var(--qp-text-muted)]">Recent Activity Timeline</h3>
				<span class="text-xs text-[var(--qp-text-muted)]">Last 5 Events</span>
			</div>
			{#if dashboard?.recent_events?.length}
				<div class="relative pl-6 border-l border-white/[0.05] space-y-6">
					{#each dashboard.recent_events.slice(0, 5) as event}
						<div class="relative">
							<!-- Timeline node -->
							<span class="absolute -left-[31px] top-1 w-2.5 h-2.5 rounded-full ring-4 ring-[var(--qp-bg)] 
								{event.event_type === 'container_start' ? 'bg-emerald-400' : 'bg-red-400'}"></span>
							<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 text-sm">
								<div class="flex items-center gap-2">
									<StatusBadge status={event.event_type === 'container_start' ? 'running' : 'stopped'} />
									<span class="text-white font-medium">{event.container_name || event.container_docker_id || 'unknown'}</span>
									<span class="text-xs text-[var(--qp-text-muted)] font-mono bg-white/5 px-1.5 py-0.5 rounded">
										{event.event_type}
									</span>
								</div>
								<span class="text-xs text-[var(--qp-text-muted)]">
									{event.timestamp ? new Date(event.timestamp).toLocaleString() : ''}
								</span>
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<EmptyState message="No recent event activity detected" />
			{/if}
		</div>
	{/if}
</div>
