<script lang="ts">
	import { onMount } from 'svelte';
	import { listStacks, startStack, stopStack, restartStack, createStack } from '$lib/api/stacks';
	import { addToast } from '$lib/stores/ui';
	import StatusBadge from '$lib/components/shared/StatusBadge.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import LoadingSkeleton from '$lib/components/shared/LoadingSkeleton.svelte';

	let stacks: any[] = $state([]);
	let loading = $state(true);
	let pendingAction = $state<string | null>(null);

	// Modal State
	let showCreateModal = $state(false);
	let newStackName = $state('');
	let newStackConfig = $state(
`version: '3.8'
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
	);
	let submitting = $state(false);

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

	function closeModal() {
		showCreateModal = false;
		newStackName = '';
		newStackConfig = `version: '3.8'
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`;
	}

	async function handleCreateStack() {
		if (!newStackName.trim()) {
			addToast('Stack name is required', 'error');
			return;
		}
		submitting = true;
		try {
			const res = await createStack(newStackName.trim(), newStackConfig);
			addToast(res.message || 'Stack created successfully', 'success');
			closeModal();
			await load();
		} catch (e: any) {
			addToast(e.message || 'Failed to create stack', 'error');
		} finally {
			submitting = false;
		}
	}

	onMount(load);
</script>

<svelte:head>
	<title>Stacks - QuickPulse</title>
</svelte:head>

<div class="flex items-center justify-between mb-6">
	<div>
		<h1 class="text-xl font-semibold text-white">Compose Stacks</h1>
		<p class="text-sm text-[var(--qp-text-muted)] mt-1">Manage your Docker Compose stacks</p>
	</div>
	<button
		class="qp-btn qp-btn-primary text-xs flex items-center gap-1.5"
		onclick={() => showCreateModal = true}
	>
		<svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
		</svg>
		Create Stack
	</button>
</div>

{#if loading}
	<LoadingSkeleton rows={3} />
{:else if stacks.length === 0}
	<EmptyState message="No compose stacks found" />
{:else}
	<div class="space-y-3">
		{#each stacks as stack (stack.name)}
			<div class="qp-card">
				<div class="flex flex-col md:flex-row md:items-center justify-between gap-3 mb-3">
					<div class="flex items-center gap-3">
						<StatusBadge status={stack.status} />
						<h3 class="text-sm font-semibold text-white">{stack.name}</h3>
						<span class="text-xs text-[var(--qp-text-muted)]">{stack.running}/{stack.total} services</span>
					</div>
					<div class="flex flex-wrap gap-2">
						<a
							class="qp-btn qp-btn-ghost text-xs flex items-center gap-1"
							href="/stacks/{stack.name}"
						>
							<svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
							</svg>
							Manage
						</a>
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

<!-- Create Stack Modal -->
{#if showCreateModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
		<div class="w-full max-w-2xl bg-[var(--qp-surface-1)] border border-[var(--qp-border)] rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
			<!-- Modal Header -->
			<div class="flex items-center justify-between px-5 py-4 border-b border-[var(--qp-border)]">
				<h3 class="text-base font-semibold text-white">Create New Compose Stack</h3>
				<button class="text-[var(--qp-text-muted)] hover:text-white transition-colors" onclick={closeModal}>
					<svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Modal Body -->
			<div class="p-5 overflow-y-auto space-y-4 flex-1">
				<div class="space-y-1">
					<label for="stack-name" class="text-xs font-semibold text-[var(--qp-text-muted)] uppercase tracking-wider">Stack Name</label>
					<input
						type="text"
						id="stack-name"
						bind:value={newStackName}
						placeholder="e.g. my-app"
						class="w-full bg-[var(--qp-surface-2)] border border-[var(--qp-border)] rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-[var(--qp-primary)] transition-colors"
					/>
				</div>

				<div class="space-y-1 flex flex-col flex-1 min-h-[300px]">
					<label for="stack-config" class="text-xs font-semibold text-[var(--qp-text-muted)] uppercase tracking-wider">docker-compose.yml</label>
					<textarea
						id="stack-config"
						bind:value={newStackConfig}
						placeholder="version: '3.8'..."
						class="w-full flex-1 bg-[var(--qp-surface-2)] border border-[var(--qp-border)] rounded-lg p-3 text-xs text-white font-mono focus:outline-none focus:border-[var(--qp-primary)] transition-colors resize-none min-h-[300px]"
					></textarea>
				</div>
			</div>

			<!-- Modal Footer -->
			<div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-[var(--qp-border)] bg-[var(--qp-surface-2)]">
				<button class="qp-btn qp-btn-ghost text-xs" onclick={closeModal}>Cancel</button>
				<button class="qp-btn qp-btn-primary text-xs" onclick={handleCreateStack} disabled={submitting}>
					{#if submitting}
						Creating...
					{:else}
						Create & Save
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}
