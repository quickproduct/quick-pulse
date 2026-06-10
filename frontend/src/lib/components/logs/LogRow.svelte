<!--
  LogRow renders a single entry as a single horizontal line. Designed to
  fit in 28px height so a virtual-list can scroll thousands of rows
  smoothly. The row truncates the message with CSS `text-overflow:ellipsis`
  rather than slicing the string, so the user can horizontally scroll if
  the table allows it (we don't, by default — they see the full message
  in the drawer).
-->
<script lang="ts">
	import type { LogEntry } from '$lib/api/logs';

	let { entry, selected = false, onClick }: {
		entry: LogEntry;
		selected?: boolean;
		onClick: (e: LogEntry) => void;
	} = $props();

	// Map level → qp-badge color class. Done once per row (not derived) so
	// Svelte doesn't track it as reactive — saves a tiny amount of CPU on
	// large lists.
	const levelClass: Record<string, string> = {
		DEBUG: 'qp-badge-neutral',
		INFO: 'qp-badge-info',
		WARN: 'qp-badge-warning',
		ERROR: 'qp-badge-danger',
		CRITICAL: 'qp-badge-danger',
	};

	// Pre-format the timestamp so the row doesn't re-do it on each scroll.
	// Locale-default keeps it readable in the user's timezone.
	const tsLabel = formatTs(entry.ts);

	function formatTs(ms: number): string {
		const d = new Date(ms);
		return (
			d.getHours().toString().padStart(2, '0') + ':' +
			d.getMinutes().toString().padStart(2, '0') + ':' +
			d.getSeconds().toString().padStart(2, '0') + '.' +
			d.getMilliseconds().toString().padStart(3, '0')
		);
	}

	// "source" column: cluster + pod/namespace for k8s, or container for docker.
	// With multi-cluster setups the cluster prefix makes the origin obvious
	// at a glance — same convention as `kubectl get pods --context`.
	const source = entry.platform === 'k8s'
		? `${entry.cluster ? entry.cluster + ' · ' : ''}${entry.namespace ?? ''}/${entry.pod ?? '?'}`
		: (entry.container ?? entry.source_id);
</script>

<button
	type="button"
	onclick={() => onClick(entry)}
	class="log-row {selected ? 'log-row-selected' : ''}"
	aria-label="View log details"
>
	<span class="log-row-ts">{tsLabel}</span>
	<span class="qp-badge {levelClass[entry.level] ?? 'qp-badge-neutral'} log-row-level">{entry.level}</span>
	<span class="log-row-platform" title={entry.platform}>{entry.platform === 'k8s' ? '☸' : '🐳'}</span>
	<span class="log-row-source" title={source}>{source}</span>
	<span class="log-row-msg">{entry.message}</span>
</button>

<style>
	.log-row {
		display: grid;
		grid-template-columns: 90px 70px 22px 220px 1fr;
		align-items: center;
		gap: 0.75rem;
		width: 100%;
		text-align: left;
		padding: 0.25rem 0.75rem;
		font-family: var(--font-family-mono);
		font-size: 0.8125rem;
		line-height: 1.4;
		border: none;
		background: transparent;
		color: var(--qp-text);
		cursor: pointer;
		min-height: 28px;
		border-bottom: 1px solid rgba(46, 51, 72, 0.4);
	}
	.log-row:hover {
		background: rgba(99, 102, 241, 0.06);
	}
	.log-row-selected {
		background: rgba(99, 102, 241, 0.14) !important;
	}
	.log-row-ts {
		color: var(--qp-text-muted);
		font-size: 0.75rem;
		white-space: nowrap;
	}
	.log-row-level {
		justify-self: start;
		font-size: 0.625rem;
		padding: 0.05rem 0.4rem;
	}
	.log-row-platform {
		text-align: center;
	}
	.log-row-source {
		color: var(--qp-text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.log-row-msg {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		color: var(--qp-text);
	}
</style>
