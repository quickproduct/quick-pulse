<script lang="ts">
	import { onMount } from 'svelte';
	import { wsManager } from '$lib/websocket/manager';
	import { liveMetrics } from '$lib/stores/metrics';
	import { getDashboard } from '$lib/api/alerts';
	import { addToast } from '$lib/stores/ui';
	import MetricCard from '$lib/components/shared/MetricCard.svelte';
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

	onMount(async () => {
		try {
			dashboard = await getDashboard();
		} catch (e: any) {
			error = e.message || 'Failed to load dashboard';
			addToast(error ?? 'Failed to load dashboard', 'error');
		} finally {
			loading = false;
		}

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
</script>

<svelte:head>
	<title>Dashboard - QuickPulse</title>
</svelte:head>

<PageHeader title="Dashboard" subtitle="System overview at a glance" />

{#if loading}
	<LoadingSkeleton rows={4} />
{:else if error}
	<div class="qp-card border-red-400/30 bg-red-400/5 text-red-400 text-sm mb-4">
		Failed to load dashboard: {error}
	</div>
{:else}
	<div class="space-y-6">
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
			<MetricCard title="CPU Usage" value={metrics?.cpu_percent?.toFixed(1) ?? '0'} unit="%" icon="cpu" />
			<MetricCard title="Memory" value={metrics?.memory_percent?.toFixed(1) ?? '0'} unit="%" icon="memory" />
			<MetricCard title="Disk" value={metrics?.disk_percent?.toFixed(1) ?? '0'} unit="%" icon="disk" />
			<MetricCard title="Load (1m)" value={metrics?.load_1m?.toFixed(2) ?? '0'} icon="load" />
		</div>

		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
			<MetricCard title="Network In" value={formatBytes(metrics?.net_bytes_recv ?? 0)} icon="network" />
			<MetricCard title="Network Out" value={formatBytes(metrics?.net_bytes_sent ?? 0)} icon="network" />
			<MetricCard title="Processes" value={String(metrics?.process_count ?? 0)} icon="containers" />
			<MetricCard title="Uptime" value={formatUptime(metrics?.uptime_seconds ?? 0)} icon="load" />
		</div>

		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<div class="qp-card">
				<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-4">Containers</h3>
				{#if dashboard?.containers}
					<div class="flex items-center gap-6">
						<div class="text-center">
							<div class="text-3xl font-bold text-white">{dashboard.containers.total ?? 0}</div>
							<div class="text-xs text-[var(--qp-text-muted)]">Total</div>
						</div>
						<div class="text-center">
							<div class="text-3xl font-bold text-green-400">{dashboard.containers.running ?? 0}</div>
							<div class="text-xs text-[var(--qp-text-muted)]">Running</div>
						</div>
						<div class="text-center">
							<div class="text-3xl font-bold text-red-400">{dashboard.containers.stopped ?? 0}</div>
							<div class="text-xs text-[var(--qp-text-muted)]">Stopped</div>
						</div>
					</div>
				{:else}
					<EmptyState message="No container data" />
				{/if}
			</div>

			<div class="qp-card">
				<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-4">Recent Events</h3>
				{#if dashboard?.recent_events?.length}
					<div class="space-y-2">
						{#each dashboard.recent_events.slice(0, 5) as event}
							<div class="flex items-center justify-between text-sm">
								<div class="flex items-center gap-2">
									<StatusBadge status={event.event_type === 'container_start' ? 'running' : 'stopped'} />
									<span class="text-white">{event.container_name || event.container_docker_id || 'unknown'}</span>
								</div>
								<span class="text-xs text-[var(--qp-text-muted)]">
									{event.timestamp ? new Date(event.timestamp).toLocaleTimeString() : ''}
								</span>
							</div>
						{/each}
					</div>
				{:else}
					<EmptyState message="No recent events" />
				{/if}
			</div>
		</div>

		{#if dashboard?.stack_summary && dashboard.stack_summary.total > 0}
			<div class="qp-card">
				<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-4">Stack Health</h3>
				<div class="flex items-center gap-6">
					<div class="text-center">
						<div class="text-2xl font-bold text-white">{dashboard.stack_summary.total ?? 0}</div>
						<div class="text-xs text-[var(--qp-text-muted)]">Stacks</div>
					</div>
					<div class="text-center">
						<div class="text-2xl font-bold text-green-400">{dashboard.stack_summary.running ?? 0}</div>
						<div class="text-xs text-[var(--qp-text-muted)]">Healthy</div>
					</div>
					<div class="text-center">
						<div class="text-2xl font-bold text-amber-400">{dashboard.stack_summary.partial ?? 0}</div>
						<div class="text-xs text-[var(--qp-text-muted)]">Partial</div>
					</div>
					<div class="text-center">
						<div class="text-2xl font-bold text-red-400">{dashboard.stack_summary.stopped ?? 0}</div>
						<div class="text-xs text-[var(--qp-text-muted)]">Stopped</div>
					</div>
				</div>
			</div>
		{/if}
	</div>
{/if}
