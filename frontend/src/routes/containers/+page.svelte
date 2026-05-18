<script lang="ts">
	import { onMount } from 'svelte';
	import { listContainers } from '$lib/api/containers';
	import { addToast } from '$lib/stores/ui';
	import { wsManager } from '$lib/websocket/manager';
	import StatusBadge from '$lib/components/shared/StatusBadge.svelte';
	import SearchBar from '$lib/components/shared/SearchBar.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import LoadingSkeleton from '$lib/components/shared/LoadingSkeleton.svelte';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';

	let containers: any[] = $state([]);
	let loading = $state(true);
	let search = $state('');
	let filter = $state('all');
	let showAll = $state(false);

	async function load() {
		loading = true;
		try {
			containers = await listContainers(showAll);
		} catch (e: any) {
			addToast(e.message || 'Failed to load containers', 'error');
		} finally {
			loading = false;
		}
	}

	let filtered = $derived(
		containers.filter((c) => {
			const matchesSearch =
				!search ||
				(c.name || '').toLowerCase().includes(search.toLowerCase()) ||
				(c.image || '').toLowerCase().includes(search.toLowerCase());
			const matchesFilter = filter === 'all' || c.status === filter;
			return matchesSearch && matchesFilter;
		})
	);

	onMount(() => {
		load();
		const unsubscribe = wsManager.onMessage('container-status', () => load());
		wsManager.connect('container-status', '/ws/container-status');
		return () => {
			unsubscribe();
			wsManager.disconnect('container-status');
		};
	});
</script>

<svelte:head>
	<title>Containers - QuickPulse</title>
</svelte:head>

<PageHeader title="Containers" subtitle="Manage your Docker containers" />

<div class="flex items-center gap-3 mb-4 flex-wrap">
	<SearchBar value={search} oninput={(v: string) => (search = v)} placeholder="Search containers..." />
	<select class="qp-input w-auto" bind:value={filter}>
		<option value="all">All Status</option>
		<option value="running">Running</option>
		<option value="exited">Stopped</option>
		<option value="paused">Paused</option>
	</select>
	<label class="flex items-center gap-2 text-sm text-[var(--qp-text-muted)] cursor-pointer">
		<input type="checkbox" bind:checked={showAll} onchange={load} class="rounded" />
		Show all
	</label>
</div>

{#if loading}
	<LoadingSkeleton rows={5} />
{:else if filtered.length === 0}
	<EmptyState message="No containers found" />
{:else}
	<div class="space-y-2">
		{#each filtered as container (container.docker_id)}
			<a
				href="/containers/{container.docker_id}"
				class="qp-card flex items-center gap-4 cursor-pointer hover:border-[var(--qp-accent)] transition-colors"
			>
				<StatusBadge status={container.status} />
				<div class="flex-1 min-w-0">
					<div class="text-sm font-medium text-white truncate">{container.name || container.docker_id}</div>
					<div class="text-xs text-[var(--qp-text-muted)] truncate">{container.image || '—'}</div>
				</div>
				<div class="text-xs text-[var(--qp-text-muted)] hidden sm:block">{container.docker_id}</div>
				<svg class="w-4 h-4 text-[var(--qp-text-muted)] shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M9 18l6-6-6-6" />
				</svg>
			</a>
		{/each}
	</div>
{/if}
