<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { inspectContainer, startContainer, stopContainer, restartContainer, getContainerLogs } from '$lib/api/containers';
	import { addToast } from '$lib/stores/ui';
	import StatusBadge from '$lib/components/shared/StatusBadge.svelte';
	import ConfirmDialog from '$lib/components/shared/ConfirmDialog.svelte';
	import LoadingSkeleton from '$lib/components/shared/LoadingSkeleton.svelte';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';

	let id: string = $derived(page.params.id || '');
	let detail: any = $state(null);
	let logs: string[] = $state([]);
	let loading = $state(true);
	let actionLoading = $state(false);
	let confirmAction = $state<{ open: boolean; action: string; label: string }>({ open: false, action: '', label: '' });

	async function load() {
		loading = true;
		try {
			detail = await inspectContainer(id);
			const logsResult = await getContainerLogs(id, 100);
			logs = logsResult?.logs || [];
		} catch (e: any) {
			addToast(e.message || 'Failed to load container', 'error');
		} finally {
			loading = false;
		}
	}

	async function executeAction(action: string) {
		actionLoading = true;
		try {
			let result: any;
			if (action === 'start') result = await startContainer(id);
			else if (action === 'stop') result = await stopContainer(id);
			else if (action === 'restart') result = await restartContainer(id);
			addToast(result?.message || `${action} succeeded`, 'success');
			await load();
		} catch (e: any) {
			addToast(e.message || `Failed to ${action} container`, 'error');
		} finally {
			actionLoading = false;
			confirmAction = { open: false, action: '', label: '' };
		}
	}

	onMount(load);
</script>

<svelte:head>
	<title>Container {id} - QuickPulse</title>
</svelte:head>

<PageHeader title="Container Detail" subtitle={id} />

{#if loading}
	<LoadingSkeleton rows={4} />
{:else if detail}
	<div class="space-y-6">
		<div class="qp-card">
			<div class="flex items-center justify-between mb-4 flex-wrap gap-3">
				<div class="flex items-center gap-3">
					<StatusBadge status={detail.State?.Status || 'unknown'} />
					<h2 class="text-lg font-semibold text-white">{detail.Name?.replace('/', '') || id}</h2>
				</div>
				<div class="flex gap-2 flex-wrap">
					<button
						class="qp-btn qp-btn-ghost"
						onclick={() => confirmAction = { open: true, action: 'start', label: 'Start' }}
						disabled={actionLoading}
					>Start</button>
					<button
						class="qp-btn qp-btn-ghost"
						onclick={() => confirmAction = { open: true, action: 'stop', label: 'Stop' }}
						disabled={actionLoading}
					>Stop</button>
					<button
						class="qp-btn qp-btn-primary"
						onclick={() => confirmAction = { open: true, action: 'restart', label: 'Restart' }}
						disabled={actionLoading}
					>
						{actionLoading ? 'Working...' : 'Restart'}
					</button>
				</div>
			</div>

			<div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
				<div>
					<span class="text-[var(--qp-text-muted)]">Image</span>
					<div class="text-white truncate">{detail.Config?.Image || '—'}</div>
				</div>
				<div>
					<span class="text-[var(--qp-text-muted)]">Created</span>
					<div class="text-white">{detail.Created ? new Date(detail.Created).toLocaleString() : '—'}</div>
				</div>
				<div>
					<span class="text-[var(--qp-text-muted)]">Status</span>
					<div class="text-white">{detail.State?.Status || '—'}</div>
				</div>
				<div>
					<span class="text-[var(--qp-text-muted)]">IP Address</span>
					<div class="text-white">{detail.NetworkSettings?.IPAddress || '—'}</div>
				</div>
			</div>
		</div>

		{#if detail.Config?.Env?.length}
			<div class="qp-card">
				<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-3">Environment Variables</h3>
				<div class="log-viewer qp-scrollbar text-xs max-h-48">
					{#each detail.Config.Env as envVar}
						<div class="log-line">
							<span class="text-blue-400">{(envVar || '').split('=')[0]}</span>={(envVar || '').split('=').slice(1).join('=')}
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<div class="qp-card">
			<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-3">Logs & Shell</h3>
			<div class="flex gap-4 mb-2 flex-wrap">
				<a href="/containers/{id}/logs" class="text-xs text-[var(--qp-accent)] hover:underline">Open live log viewer &rarr;</a>
				<a href="/containers/{id}/terminal" class="text-xs text-[var(--qp-accent)] hover:underline">Open interactive terminal &rarr;</a>
			</div>
			<div class="log-viewer qp-scrollbar text-xs">
				{#if logs.length}
					{#each logs as line}
						<div class="log-line">{line}</div>
					{/each}
				{:else}
					<div class="text-[var(--qp-text-muted)]">No logs available</div>
				{/if}
			</div>
		</div>
	</div>
{:else}
	<div class="qp-card text-[var(--qp-text-muted)] text-sm text-center py-8">
		Container not found or failed to load.
	</div>
{/if}

<ConfirmDialog
	open={confirmAction.open}
	title="Confirm {confirmAction.label}"
	message="Are you sure you want to {confirmAction.label} container {id}?"
	confirmLabel={confirmAction.label || 'Confirm'}
	onconfirm={() => executeAction(confirmAction.action)}
	oncancel={() => confirmAction = { open: false, action: '', label: '' }}
/>
