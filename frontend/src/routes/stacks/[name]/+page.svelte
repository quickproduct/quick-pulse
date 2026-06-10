<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getStack, getStackConfig, saveStackConfig, startStack, stopStack, restartStack, deployStack } from '$lib/api/stacks';
	import { addToast } from '$lib/stores/ui';
	import StatusBadge from '$lib/components/shared/StatusBadge.svelte';
	import LoadingSkeleton from '$lib/components/shared/LoadingSkeleton.svelte';

	let name: string = $derived(page.params.name || '');

	let stack: any = $state(null);
	let config: string = $state('');
	let loading = $state(true);
	let activeTab = $state<'services' | 'editor'>('services');

	let saving = $state(false);
	let deploying = $state(false);
	let pendingAction = $state<string | null>(null);

	// Streaming logs state
	let deployLogs = $state<string[]>([]);
	let showLogsModal = $state(false);
	let logsContainer: HTMLDivElement | undefined = $state();

	async function load() {
		loading = true;
		try {
			const [stackRes, configRes] = await Promise.all([
				getStack(name).catch(() => null),
				getStackConfig(name).catch(() => ({ name, config: '' }))
			]);
			stack = stackRes;
			config = configRes.config;
		} catch (e: any) {
			addToast(e.message || 'Failed to load stack details', 'error');
		} finally {
			loading = false;
		}
	}

	async function handleAction(action: 'start' | 'stop' | 'restart') {
		pendingAction = action;
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

	async function handleSave() {
		saving = true;
		try {
			const res = await saveStackConfig(name, config);
			addToast(res.message || 'Configuration saved', 'success');
			await load();
		} catch (e: any) {
			addToast(e.message || 'Failed to save configuration', 'error');
		} finally {
			saving = false;
		}
	}

	async function handleDeploy() {
		deploying = true;
		showLogsModal = true;
		deployLogs = [];
		try {
			// First save configuration changes
			await saveStackConfig(name, config);
			
			// Start deployment stream
			await deployStack(name, (chunk) => {
				deployLogs.push(chunk);
				// Scroll to bottom
				if (logsContainer) {
					setTimeout(() => {
						if (logsContainer) {
							logsContainer.scrollTop = logsContainer.scrollHeight;
						}
					}, 30);
				}
			});
			addToast('Deployment completed', 'success');
			await load();
		} catch (e: any) {
			addToast(e.message || 'Deployment failed', 'error');
		} finally {
			deploying = false;
		}
	}

	onMount(load);
</script>

<svelte:head>
	<title>{name} - Stack Details - QuickPulse</title>
</svelte:head>

<div class="max-w-7xl mx-auto space-y-6">
	<!-- Breadcrumbs and Header -->
	<div>
		<a href="/stacks" class="text-xs text-[var(--qp-accent, #3b82f6)] hover:underline flex items-center gap-1">
			&larr; Back to Stacks
		</a>
		{#if stack}
			<div class="flex flex-col md:flex-row md:items-center justify-between gap-4 mt-2">
				<div class="flex items-center gap-3">
					<StatusBadge status={stack.status} />
					<h1 class="text-3xl font-bold text-white tracking-tight">{name}</h1>
					<span class="text-xs text-[var(--qp-text-muted)]">({stack.running}/{stack.total} services running)</span>
				</div>
				<div class="flex gap-2 flex-wrap">
					<button
						class="qp-btn qp-btn-ghost text-xs"
						onclick={() => handleAction('start')}
						disabled={pendingAction !== null || deploying}
					>
						Start
					</button>
					<button
						class="qp-btn qp-btn-ghost text-xs"
						onclick={() => handleAction('stop')}
						disabled={pendingAction !== null || deploying}
					>
						Stop
					</button>
					<button
						class="qp-btn qp-btn-primary text-xs"
						onclick={() => handleAction('restart')}
						disabled={pendingAction !== null || deploying}
					>
						{pendingAction === 'restart' ? 'Restarting...' : 'Restart'}
					</button>
				</div>
			</div>
			<p class="text-xs text-[var(--qp-text-muted)] font-mono mt-1">{stack.project_dir}</p>
		{:else}
			<div class="flex flex-col md:flex-row md:items-center justify-between gap-4 mt-2">
				<div class="flex items-center gap-3">
					<StatusBadge status="stopped" />
					<h1 class="text-3xl font-bold text-white tracking-tight">{name}</h1>
					<span class="text-xs text-[var(--qp-text-muted)]">(stopped)</span>
				</div>
				<div class="flex gap-2 flex-wrap">
					<button
						class="qp-btn qp-btn-primary text-xs"
						onclick={() => activeTab = 'editor'}
					>
						Configure Stack
					</button>
				</div>
			</div>
		{/if}
	</div>

	{#if loading}
		<LoadingSkeleton rows={4} />
	{:else}
		<!-- Tabs -->
		<div class="border-b border-white/10">
			<nav class="flex gap-8">
				<button
					onclick={() => activeTab = 'services'}
					class="pb-4 text-sm font-medium transition-all relative {activeTab === 'services' ? 'text-white' : 'text-[var(--qp-text-muted)] hover:text-white'}"
				>
					Services
					{#if activeTab === 'services'}
						<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500 shadow-[0_0_10px_rgba(59,130,246,0.5)]"></div>
					{/if}
				</button>
				<button
					onclick={() => activeTab = 'editor'}
					class="pb-4 text-sm font-medium transition-all relative {activeTab === 'editor' ? 'text-white' : 'text-[var(--qp-text-muted)] hover:text-white'}"
				>
					YAML Config Editor
					{#if activeTab === 'editor'}
						<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500 shadow-[0_0_10px_rgba(59,130,246,0.5)]"></div>
					{/if}
				</button>
			</nav>
		</div>

		<!-- Tab Contents -->
		{#if activeTab === 'services'}
			{#if stack && stack.services && stack.services.length > 0}
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					{#each stack.services as svc}
						<div class="qp-card p-4 flex flex-col justify-between min-h-[120px]">
							<div>
								<div class="flex items-center justify-between mb-2 gap-2">
									<h3 class="text-sm font-semibold text-white truncate max-w-[70%]" title={svc.name}>{svc.name}</h3>
									<StatusBadge status={svc.status} />
								</div>
								<div class="text-xs font-mono text-[var(--qp-text-muted)] mt-1">
									Container ID: <span class="text-white font-mono">{svc.container_id || '—'}</span>
								</div>
							</div>
							
							{#if svc.container_id}
								<div class="flex justify-end border-t border-white/5 pt-3 mt-4">
									<a
										href="/containers/{svc.container_id}/terminal"
										class="text-xs text-[var(--qp-accent, #3b82f6)] hover:underline flex items-center gap-1 font-medium"
									>
										<svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
										</svg>
										Terminal Shell &rarr;
									</a>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{:else}
				<div class="qp-card p-12 text-center text-[var(--qp-text-muted)]">
					No running or configured services found for this stack. Go to "YAML Config Editor" to save and deploy the stack configuration.
				</div>
			{/if}
		{:else if activeTab === 'editor'}
			<div class="space-y-4">
				<div class="qp-card p-4 bg-[#0d0e12] border border-[var(--qp-border)] rounded-lg flex flex-col">
					<div class="flex items-center justify-between mb-3">
						<span class="text-xs font-semibold text-[var(--qp-text-muted)] uppercase tracking-wider font-mono">docker-compose.yml</span>
						<span class="text-xs text-[var(--qp-text-muted)] font-mono">YAML Format</span>
					</div>
					<textarea
						bind:value={config}
						placeholder="version: '3.8'&#10;services:&#10;  web:&#10;    image: nginx:alpine"
						class="w-full h-[50vh] bg-[#090a0f] border border-white/5 rounded-lg p-4 text-xs text-white font-mono focus:outline-none focus:border-[var(--qp-primary)] transition-colors resize-none leading-relaxed"
						spellcheck="false"
					></textarea>
				</div>
				<div class="flex justify-end gap-3">
					<button
						class="qp-btn qp-btn-ghost text-xs"
						onclick={handleSave}
						disabled={saving || deploying}
					>
						{saving ? 'Saving...' : 'Save Configuration'}
					</button>
					<button
						class="qp-btn qp-btn-primary text-xs flex items-center gap-1.5"
						onclick={handleDeploy}
						disabled={saving || deploying}
					>
						<svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
						</svg>
						Save & Deploy Stack
					</button>
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- Deployment Logs Modal -->
{#if showLogsModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm">
		<div class="w-full max-w-4xl bg-[var(--qp-surface-1)] border border-[var(--qp-border)] rounded-xl shadow-2xl overflow-hidden flex flex-col h-[75vh]">
			<!-- Header -->
			<div class="flex items-center justify-between px-5 py-4 border-b border-[var(--qp-border)]">
				<div class="flex items-center gap-2">
					{#if deploying}
						<div class="w-2 h-2 rounded-full bg-blue-500 animate-ping"></div>
					{/if}
					<h3 class="text-base font-semibold text-white">Stack Deployment Progress</h3>
				</div>
				{#if !deploying}
					<button class="text-[var(--qp-text-muted)] hover:text-white transition-colors" onclick={() => showLogsModal = false}>
						<svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				{/if}
			</div>

			<!-- Logs Terminal -->
			<div 
				bind:this={logsContainer}
				class="flex-1 p-5 bg-[#090a0f] font-mono text-xs overflow-y-auto text-gray-300 space-y-1.5 selection:bg-blue-500/30 select-all scroll-smooth"
			>
				{#each deployLogs as line}
					<div class="whitespace-pre-wrap leading-relaxed">{line}</div>
				{/each}
				
				{#if deployLogs.length === 0}
					<div class="text-[var(--qp-text-muted)] italic">Initializing docker compose execution...</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="flex items-center justify-between px-5 py-4 border-t border-[var(--qp-border)] bg-[var(--qp-surface-2)]">
				<span class="text-xs text-[var(--qp-text-muted)] font-mono">
					{#if deploying}
						Running `docker compose up -d`...
					{:else}
						Process exited.
					{/if}
				</span>
				<button 
					class="qp-btn qp-btn-ghost text-xs" 
					onclick={() => showLogsModal = false}
					disabled={deploying}
				>
					Close Viewer
				</button>
			</div>
		</div>
	</div>
{/if}
