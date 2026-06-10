<script lang="ts">
	let { status = 'unknown' } = $props();

	const statusConfig: Record<string, { label: string; class: string }> = {
		running: { label: 'Running', class: 'qp-badge-success' },
		exited: { label: 'Stopped', class: 'qp-badge-danger' },
		stopped: { label: 'Stopped', class: 'qp-badge-danger' },
		paused: { label: 'Paused', class: 'qp-badge-warning' },
		restarting: { label: 'Restarting', class: 'qp-badge-warning' },
		dead: { label: 'Dead', class: 'qp-badge-danger' },
		created: { label: 'Created', class: 'qp-badge-info' },
		partial: { label: 'Partial', class: 'qp-badge-warning' },
		unknown: { label: 'Unknown', class: 'qp-badge-neutral' },
	};

	let config = $derived(statusConfig[status] || statusConfig.unknown);
</script>

<span class="qp-badge {config.class}">
	{#if status === 'running'}
		<span class="w-1.5 h-1.5 rounded-full bg-green-400 pulse-dot mr-1.5"></span>
	{/if}
	{config.label}
</span>
