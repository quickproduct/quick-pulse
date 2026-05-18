<script lang="ts">
	import { onMount } from 'svelte';
	import * as k8s from '$lib/api/kubernetes';
	import ClusterOverview from '$lib/components/kubernetes/ClusterOverview.svelte';
	import PodRow from '$lib/components/kubernetes/PodRow.svelte';
	import DeploymentCard from '$lib/components/kubernetes/DeploymentCard.svelte';
	import ServiceRow from '$lib/components/kubernetes/ServiceRow.svelte';
	import EventsList from '$lib/components/kubernetes/EventsList.svelte';

	let overview = $state<k8s.ClusterOverview | null>(null);
	let pods = $state<k8s.KubePod[]>([]);
	let deployments = $state<k8s.KubeDeployment[]>([]);
	let services = $state<k8s.KubeService[]>([]);
	let namespaces = $state<string[]>([]);
	let events = $state<k8s.KubeEvent[]>([]);
	let selectedNamespace = $state<string>('');
	let activeTab = $state<'pods' | 'deployments' | 'services' | 'events'>('pods');
	let loading = $state(true);

	async function fetchData() {
		loading = true;
		try {
			const nsFilter = selectedNamespace || undefined;
			[overview, pods, deployments, services, namespaces, events] = await Promise.all([
				k8s.getClusterOverview(),
				k8s.getPods(nsFilter),
				k8s.getDeployments(nsFilter),
				k8s.getServices(nsFilter),
				k8s.getNamespaces(),
				k8s.getEvents(nsFilter)
			]);
		} catch (err) {
			console.error('Failed to fetch K8s data', err);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchData();
		const interval = setInterval(fetchData, 10000); // Refresh every 10s
		return () => clearInterval(interval);
	});

	$effect(() => {
		if (selectedNamespace !== undefined) {
			fetchData();
		}
	});
</script>

<div class="p-6 max-w-7xl mx-auto">
	<div class="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8">
		<div>
			<h1 class="text-3xl font-bold text-white tracking-tight">Kubernetes Dashboard</h1>
			<p class="text-[var(--qp-text-muted)] mt-1">Real-time monitoring of cluster resources and health.</p>
		</div>

		<div class="flex items-center gap-3">
			<select 
				bind:value={selectedNamespace}
				class="bg-white/5 border border-white/10 text-white text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block p-2.5 font-mono"
			>
				<option value="">All Namespaces</option>
				{#each namespaces as ns}
					<option value={ns}>{ns}</option>
				{/each}
			</select>

			<button 
				onclick={fetchData}
				class="qp-button-secondary py-2.5 px-4 flex items-center gap-2"
				disabled={loading}
			>
				<svg class="w-4 h-4 {loading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				Refresh
			</button>
		</div>
	</div>

	{#if overview}
		<ClusterOverview {overview} />
	{/if}

	<!-- Tabs -->
	<div class="mb-6 border-b border-white/10">
		<nav class="flex gap-8">
			{#each ['pods', 'deployments', 'services', 'events'] as tab}
				<button
					onclick={() => activeTab = tab as any}
					class="pb-4 text-sm font-medium transition-all relative {activeTab === tab ? 'text-white' : 'text-[var(--qp-text-muted)] hover:text-white'}"
				>
					{tab.charAt(0).toUpperCase() + tab.slice(1)}
					{#if activeTab === tab}
						<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500 shadow-[0_0_10px_rgba(59,130,246,0.5)]"></div>
					{/if}
				</button>
			{/each}
		</nav>
	</div>

	{#if loading && !overview}
		<div class="flex items-center justify-center py-20">
			<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-white"></div>
		</div>
	{:else}
		<div class="space-y-6">
			{#if activeTab === 'pods'}
				<div class="qp-card overflow-hidden">
					<table class="w-full text-left border-collapse">
						<thead>
							<tr class="text-[var(--qp-text-muted)] text-[10px] uppercase tracking-widest font-bold border-b border-white/5 bg-white/[0.02]">
								<th class="px-4 py-3">Name</th>
								<th class="px-4 py-3">Namespace</th>
								<th class="px-4 py-3">Status</th>
								<th class="px-4 py-3">Ready</th>
								<th class="px-4 py-3 text-center">Restarts</th>
								<th class="px-4 py-3">Node</th>
								<th class="px-4 py-3">Age</th>
							</tr>
						</thead>
						<tbody>
							{#each pods as pod}
								<PodRow {pod} />
							{/each}
							{#if pods.length === 0}
								<tr>
									<td colspan="7" class="px-4 py-8 text-center text-[var(--qp-text-muted)]">No pods found in this namespace</td>
								</tr>
							{/if}
						</tbody>
					</table>
				</div>
			{:else if activeTab === 'deployments'}
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					{#each deployments as deployment}
						<DeploymentCard {deployment} />
					{/each}
					{#if deployments.length === 0}
						<div class="col-span-full qp-card p-12 text-center text-[var(--qp-text-muted)]">
							No deployments found in this namespace
						</div>
					{/if}
				</div>
			{:else if activeTab === 'services'}
				<div class="qp-card overflow-hidden">
					<table class="w-full text-left border-collapse">
						<thead>
							<tr class="text-[var(--qp-text-muted)] text-[10px] uppercase tracking-widest font-bold border-b border-white/5 bg-white/[0.02]">
								<th class="px-4 py-3">Name</th>
								<th class="px-4 py-3">Namespace</th>
								<th class="px-4 py-3">Type</th>
								<th class="px-4 py-3">Cluster IP</th>
								<th class="px-4 py-3">External IP</th>
								<th class="px-4 py-3">Ports</th>
							</tr>
						</thead>
						<tbody>
							{#each services as service}
								<ServiceRow {service} />
							{/each}
							{#if services.length === 0}
								<tr>
									<td colspan="6" class="px-4 py-8 text-center text-[var(--qp-text-muted)]">No services found in this namespace</td>
								</tr>
							{/if}
						</tbody>
					</table>
				</div>
			{:else if activeTab === 'events'}
				<EventsList {events} />
			{/if}
		</div>
	{/if}
</div>

<style>
	select {
		appearance: none;
		background-image: url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e");
		background-repeat: no-repeat;
		background-position: right 0.75rem center;
		background-size: 1em;
		padding-right: 2.5rem;
	}
</style>
