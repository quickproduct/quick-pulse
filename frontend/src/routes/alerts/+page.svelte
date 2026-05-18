<script lang="ts">
	import { onMount } from 'svelte';
	import { listAlerts, listAlertRules, acknowledgeAlert, createAlertRule, deleteAlertRule } from '$lib/api/alerts';
	import { addToast } from '$lib/stores/ui';
	import StatusBadge from '$lib/components/shared/StatusBadge.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import LoadingSkeleton from '$lib/components/shared/LoadingSkeleton.svelte';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';

	let alerts: any[] = $state([]);
	let rules: any[] = $state([]);
	let loading = $state(true);
	let showCreateForm = $state(false);
	let creating = $state(false);
	let acknowledgingId = $state<string | null>(null);
	let newRule = $state({ metric_type: 'cpu', threshold: 90, operator: 'gte', duration_seconds: 60 });

	async function load() {
		try {
			[alerts, rules] = await Promise.all([listAlerts(), listAlertRules()]);
		} catch (e: any) {
			addToast(e.message, 'error');
		} finally {
			loading = false;
		}
	}

	async function handleAcknowledge(id: string) {
		acknowledgingId = id;
		try {
			await acknowledgeAlert(id);
			addToast('Alert acknowledged', 'success');
			await load();
		} catch (e: any) {
			addToast(e.message || 'Failed to acknowledge alert', 'error');
		} finally {
			acknowledgingId = null;
		}
	}

	async function handleCreateRule() {
		if (newRule.threshold < 0) { addToast('Threshold must be 0 or greater', 'error'); return; }
		if (newRule.duration_seconds < 1) { addToast('Duration must be at least 1 second', 'error'); return; }
		creating = true;
		try {
			await createAlertRule(newRule);
			addToast('Alert rule created', 'success');
			showCreateForm = false;
			newRule = { metric_type: 'cpu', threshold: 90, operator: 'gte', duration_seconds: 60 };
			await load();
		} catch (e: any) {
			addToast(e.message || 'Failed to create rule', 'error');
		} finally {
			creating = false;
		}
	}

	async function handleDeleteRule(id: string) {
		try {
			await deleteAlertRule(id);
			addToast('Rule deleted', 'success');
			await load();
		} catch (e: any) {
			addToast(e.message, 'error');
		}
	}

	function getSeverityColor(severity: string): string {
		if (severity === 'critical') return 'qp-badge-danger';
		if (severity === 'warning') return 'qp-badge-warning';
		return 'qp-badge-info';
	}

	onMount(load);
</script>

<svelte:head>
	<title>Alerts - QuickPulse</title>
</svelte:head>

<PageHeader title="Alerts" subtitle="Monitor and configure alert rules" />

{#if loading}
	<LoadingSkeleton rows={4} />
{:else}
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
		<div>
			<h2 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-3">Active Alerts</h2>
			{#if alerts.length === 0}
				<EmptyState message="No active alerts" />
			{:else}
				<div class="space-y-2">
					{#each alerts as alert (alert.id)}
						<div class="qp-card flex items-center gap-3">
							<span class="qp-badge {getSeverityColor(alert.severity)}">{alert.severity}</span>
							<div class="flex-1 min-w-0">
								<p class="text-sm text-white">{alert.message}</p>
								<p class="text-xs text-[var(--qp-text-muted)]">{alert.created_at ? new Date(alert.created_at).toLocaleString() : ''}</p>
							</div>
							<button
								class="qp-btn qp-btn-ghost text-xs"
								onclick={() => handleAcknowledge(alert.id)}
								disabled={acknowledgingId === alert.id}
							>{acknowledgingId === alert.id ? 'Acknowledging...' : 'Acknowledge'}</button>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<div>
			<div class="flex items-center justify-between mb-3">
				<h2 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide">Alert Rules</h2>
				<button class="qp-btn qp-btn-primary text-xs" onclick={() => (showCreateForm = !showCreateForm)}>
					{showCreateForm ? 'Cancel' : '+ Add Rule'}
				</button>
			</div>

			{#if showCreateForm}
				<div class="qp-card mb-4 space-y-3">
					<div>
						<label class="block text-xs text-[var(--qp-text-muted)] mb-1">Metric</label>
						<select class="qp-input" bind:value={newRule.metric_type}>
							<option value="cpu">CPU</option>
							<option value="memory">Memory</option>
							<option value="disk">Disk</option>
							<option value="load">Load</option>
						</select>
					</div>
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label class="block text-xs text-[var(--qp-text-muted)] mb-1">Threshold</label>
							<input type="number" class="qp-input" bind:value={newRule.threshold} min="0" step="0.1" />
						</div>
						<div>
							<label class="block text-xs text-[var(--qp-text-muted)] mb-1">Operator</label>
							<select class="qp-input" bind:value={newRule.operator}>
							<option value="gte">&gt;=</option>
							<option value="gt">&gt;</option>
							<option value="lte">&lt;=</option>
							<option value="lt">&lt;</option>
							<option value="eq">=</option>
							</select>
						</div>
					</div>
					<div>
						<label class="block text-xs text-[var(--qp-text-muted)] mb-1">Duration (seconds)</label>
						<input type="number" class="qp-input" bind:value={newRule.duration_seconds} min="1" step="1" />
					</div>
					<button class="qp-btn qp-btn-primary w-full" onclick={handleCreateRule} disabled={creating}>
						{creating ? 'Creating...' : 'Create Rule'}
					</button>
				</div>
			{/if}

			{#if rules.length === 0}
				<EmptyState message="No alert rules configured" />
			{:else}
				<div class="space-y-2">
					{#each rules as rule (rule.id)}
						<div class="qp-card flex items-center gap-3">
							<div class="flex-1">
								<p class="text-sm text-white">
									{rule.metric_type.toUpperCase()} {rule.operator} {rule.threshold}%
								</p>
								<p class="text-xs text-[var(--qp-text-muted)]">Duration: {rule.duration_seconds}s | {rule.enabled ? 'Enabled' : 'Disabled'}</p>
							</div>
							<button class="qp-btn qp-btn-danger text-xs" onclick={() => handleDeleteRule(rule.id)}>Delete</button>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>
{/if}
