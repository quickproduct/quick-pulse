<script lang="ts">
	import { onMount } from 'svelte';
	import { listStacks, startStack, stopStack, restartStack } from '$lib/api/stacks';
	import { addToast } from '$lib/stores/ui';
	import StatusBadge from '$lib/components/shared/StatusBadge.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import LoadingSkeleton from '$lib/components/shared/LoadingSkeleton.svelte';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';

	let stacks: any[] = $state([]);
	let loading = $state(true);
	let pendingAction = $state<string | null>(null);

	async function load() {
		try {
			stacks = await listStacks();
		} catch (e: any) {
			addToast(e.message || 'Failed to load stacks', 'error');
		} finally {
			loading = false;
		}
	}

	async function handleAction(name: string, action: 'start' | 'stop' | 'restart') {
		pendingAction = `${name}:${action}`;
		try {
			const fn = action === 'start' ? startStack : action === 'stop' ? stopStack : restartStack;
			const result = await fn(name);
			addToast(result.message || `${action} succeeded`, 'success');
			await load();
		} catch (e: any) {
			addToast(e.message || `Failed to ${action} stack`, 'error');
		} finally {
			pendingAction = null;
		}
	}

	onMount(load);
</script>

<svelte:head>
	<title>Stacks - QuickPulse</title>
</svelte:head>

<PageHeader title="Compose Stacks" subtitle="Manage your Docker Compose stacks" />

{#if loading}
	<LoadingSkeleton rows={3} />
{:else if stacks.length === 0}
	<EmptyState message="No compose stacks found" />
{:else}
	<div class="space-y-3">
		{#each stacks as stack (stack.name)}
			<div class="qp-card">
				<div class="flex items-center justify-between mb-3">
					<div class="flex items-center gap-3">
						<StatusBadge status={stack.status} />
						<h3 class="text-sm font-semibold text-white">{stack.name}</h3>
						<span class="text-xs text-[var(--qp-text-muted)]">{stack.running}/{stack.total} services</span>
					</div>
					<div class="flex gap-2">
						<button
							class="qp-btn qp-btn-ghost text-xs"
							onclick={() => handleAction(stack.name, 'start')}
							disabled={pendingAction !== null}
						>Start</button>
						<button
							class="qp-btn qp-btn-ghost text-xs"
							onclick={() => handleAction(stack.name, 'stop')}
							disabled={pendingAction !== null}
						>Stop</button>
						<button
							class="qp-btn qp-btn-primary text-xs"
							onclick={() => handleAction(stack.name, 'restart')}
							disabled={pendingAction !== null}
						>{pendingAction === `${stack.name}:restart` ? 'Working...' : 'Restart'}</button>
					</div>
				</div>
				{#if stack.services?.length}
					<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
						{#each stack.services as svc}
							<div class="flex items-center gap-2 text-xs bg-[var(--qp-surface-2)] rounded-lg px-3 py-2">
								<StatusBadge status={svc.status} />
								<span class="text-white">{svc.name}</span>
								<span class="text-[var(--qp-text-muted)] ml-auto">{svc.container_id}</span>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}
