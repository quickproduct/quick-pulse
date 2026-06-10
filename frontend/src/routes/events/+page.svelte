<script lang="ts">
	import { onMount } from 'svelte';
	import { getEvents } from '$lib/api/events';
	import { addToast } from '$lib/stores/ui';
	import { wsManager } from '$lib/websocket/manager';
	import StatusBadge from '$lib/components/shared/StatusBadge.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import LoadingSkeleton from '$lib/components/shared/LoadingSkeleton.svelte';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';

	interface Event {
		id: string;
		container_docker_id: string | null;
		container_name: string | null;
		event_type: string;
		timestamp: string | null;
		metadata: any;
	}

	let events: Event[] = $state([]);
	let loading = $state(true);
	let live = $state(true);

	function getEventStatus(type: string): string {
		if (type.includes('start')) return 'running';
		if (type.includes('stop') || type.includes('die')) return 'stopped';
		if (type.includes('restart')) return 'restarting';
		return 'unknown';
	}

	function formatEventType(type: string): string {
		return (type || '').replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
	}

	onMount(() => {
		async function init() {
			try {
				events = await getEvents(100);
			} catch (e: any) {
				addToast(e.message || 'Failed to load events', 'error');
			} finally {
				loading = false;
			}
		}
		init();

		const unsubscribe = wsManager.onMessage('events', (data: Event) => {
			if (live) {
				events = [data, ...events].slice(0, 200);
			}
		});
		wsManager.connect('events', '/ws/events');

		return () => {
			unsubscribe();
			wsManager.disconnect('events');
		};
	});
</script>

<svelte:head>
	<title>Events - QuickPulse</title>
</svelte:head>

<PageHeader title="Events" subtitle="Live Docker event stream" />

<div class="flex items-center gap-3 mb-4">
	<label class="flex items-center gap-2 text-sm text-[var(--qp-text-muted)] cursor-pointer">
		<input type="checkbox" bind:checked={live} class="rounded" />
		Live updates
	</label>
	{#if live}
		<span class="flex items-center gap-1.5 text-xs text-green-400">
			<span class="w-1.5 h-1.5 rounded-full bg-green-400 pulse-dot"></span>
			Live
		</span>
	{/if}
</div>

{#if loading}
	<LoadingSkeleton rows={5} />
{:else if events.length === 0}
	<EmptyState message="No events recorded" />
{:else}
	<div class="space-y-1">
		{#each events as event (event.id)}
			<div class="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-[var(--qp-surface-2)] text-sm transition-colors">
				<StatusBadge status={getEventStatus(event.event_type)} />
				<span class="text-white font-medium min-w-0 truncate">{event.container_name || event.container_docker_id || 'unknown'}</span>
				<span class="text-[var(--qp-text-muted)] text-xs whitespace-nowrap">{formatEventType(event.event_type)}</span>
				<span class="ml-auto text-xs text-[var(--qp-text-muted)] shrink-0">
					{event.timestamp ? new Date(event.timestamp).toLocaleTimeString() : ''}
				</span>
			</div>
		{/each}
	</div>
{/if}
