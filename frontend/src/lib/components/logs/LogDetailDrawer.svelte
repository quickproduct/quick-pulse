<!--
  Right-side drawer showing one log entry's full detail. Slides in from the
  right with a CSS transform; closes on ESC, on backdrop click, or via the
  close button. The drawer is mounted by the parent into the page; we
  control visibility via a single `open` prop so animation cleanup is
  trivial.

  Meta JSON is rendered as a collapsible <pre> for now — sufficient for
  the MVP. A proper tree renderer can be added later without breaking the
  surface.
-->
<script lang="ts">
	import type { LogEntry } from '$lib/api/logs';

	let { entry, onClose }: { entry: LogEntry | null; onClose: () => void } = $props();

	$effect(() => {
		if (!entry) return;
		const onKey = (e: KeyboardEvent) => {
			if (e.key === 'Escape') onClose();
		};
		window.addEventListener('keydown', onKey);
		return () => window.removeEventListener('keydown', onKey);
	});

	function formatTs(ms: number): string {
		return new Date(ms).toISOString();
	}

	// `meta` is a JSON string; pretty-print if parseable, else show raw.
	const prettyMeta = $derived.by(() => {
		if (!entry?.meta) return '';
		try {
			return JSON.stringify(JSON.parse(entry.meta), null, 2);
		} catch {
			return entry.meta;
		}
	});

	function copyMessage() {
		if (!entry) return;
		navigator.clipboard?.writeText(entry.message);
	}

	function downloadJSON() {
		if (!entry) return;
		const blob = new Blob([JSON.stringify(entry, null, 2)], {
			type: 'application/json'
		});
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `log-${entry.id}.json`;
		a.click();
		URL.revokeObjectURL(url);
	}

	const levelClass: Record<string, string> = {
		DEBUG: 'qp-badge-neutral',
		INFO: 'qp-badge-info',
		WARN: 'qp-badge-warning',
		ERROR: 'qp-badge-danger',
		CRITICAL: 'qp-badge-danger',
	};
</script>

{#if entry}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="backdrop" onclick={onClose}></div>
	<aside class="drawer">
		<header class="drawer-header">
			<div class="left">
				<span class="qp-badge {levelClass[entry.level] ?? 'qp-badge-neutral'}">{entry.level}</span>
				<span class="ts">{formatTs(entry.ts)}</span>
			</div>
			<button class="qp-btn qp-btn-ghost close" onclick={onClose} aria-label="Close">✕</button>
		</header>

		<section class="section">
			<div class="meta-grid">
				<div><label>Platform</label><span>{entry.platform}</span></div>
				{#if entry.cluster}<div><label>Cluster</label><span class="mono">{entry.cluster}</span></div>{/if}
				<div><label>Source ID</label><span class="mono">{entry.source_id}</span></div>
				{#if entry.container}<div><label>Container</label><span class="mono">{entry.container}</span></div>{/if}
				{#if entry.namespace}<div><label>Namespace</label><span class="mono">{entry.namespace}</span></div>{/if}
				{#if entry.pod}<div><label>Pod</label><span class="mono">{entry.pod}</span></div>{/if}
				{#if entry.service}<div><label>Service</label><span class="mono">{entry.service}</span></div>{/if}
				{#if entry.host}<div><label>Host</label><span class="mono">{entry.host}</span></div>{/if}
				{#if entry.env}<div><label>Environment</label><span class="mono">{entry.env}</span></div>{/if}
				{#if entry.trace_id}<div><label>Trace ID</label><span class="mono">{entry.trace_id}</span></div>{/if}
			</div>
		</section>

		<section class="section">
			<div class="section-header">
				<h3>Message</h3>
				<div class="actions">
					<button class="qp-btn qp-btn-ghost xs" onclick={copyMessage}>Copy</button>
					<button class="qp-btn qp-btn-ghost xs" onclick={downloadJSON}>Download JSON</button>
				</div>
			</div>
			<pre class="message">{entry.message}</pre>
		</section>

		{#if prettyMeta}
			<section class="section">
				<h3>Meta</h3>
				<pre class="meta">{prettyMeta}</pre>
			</section>
		{/if}

		<section class="section">
			<h3>Open source viewer</h3>
			<div class="links">
				{#if entry.platform === 'docker' && entry.container}
					<a class="qp-btn qp-btn-ghost xs" href="/containers/{entry.source_id.replace('docker:', '').slice(0, 12)}/logs">
						Container logs →
					</a>
				{:else if entry.platform === 'k8s' && entry.pod && entry.namespace}
					<a class="qp-btn qp-btn-ghost xs" href="/kubernetes/pods/{entry.namespace}/{entry.pod}/logs">
						Pod logs →
					</a>
				{/if}
			</div>
		</section>
	</aside>
{/if}

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.45);
		z-index: 40;
		animation: fade-in 0.15s ease-out;
	}
	.drawer {
		position: fixed;
		top: 0;
		right: 0;
		bottom: 0;
		width: min(600px, 100vw);
		background: var(--qp-surface);
		border-left: 1px solid var(--qp-border);
		z-index: 50;
		display: flex;
		flex-direction: column;
		overflow-y: auto;
		animation: slide-in 0.18s ease-out;
	}
	@keyframes fade-in {
		from { opacity: 0; }
		to { opacity: 1; }
	}
	@keyframes slide-in {
		from { transform: translateX(20px); opacity: 0; }
		to { transform: translateX(0); opacity: 1; }
	}
	.drawer-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem 1.25rem;
		border-bottom: 1px solid var(--qp-border);
		position: sticky;
		top: 0;
		background: var(--qp-surface);
		z-index: 1;
	}
	.left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.ts {
		font-family: var(--font-family-mono);
		font-size: 0.8125rem;
		color: var(--qp-text-muted);
	}
	.close {
		padding: 0.25rem 0.5rem;
	}
	.section {
		padding: 1rem 1.25rem;
		border-bottom: 1px solid var(--qp-border);
	}
	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.5rem;
	}
	.section h3 {
		font-size: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--qp-text-muted);
		margin: 0 0 0.5rem 0;
	}
	.actions {
		display: flex;
		gap: 0.375rem;
	}
	.xs {
		padding: 0.25rem 0.5rem;
		font-size: 0.7rem;
	}
	.meta-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.625rem 1rem;
	}
	.meta-grid label {
		display: block;
		font-size: 0.625rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--qp-text-muted);
		margin-bottom: 0.125rem;
	}
	.meta-grid span {
		font-size: 0.8125rem;
		color: var(--qp-text);
	}
	.mono {
		font-family: var(--font-family-mono);
	}
	.message,
	.meta {
		background: #0d0f18;
		border: 1px solid var(--qp-border);
		border-radius: 0.375rem;
		padding: 0.75rem;
		font-family: var(--font-family-mono);
		font-size: 0.8125rem;
		color: var(--qp-text);
		white-space: pre-wrap;
		word-break: break-word;
		max-height: 50vh;
		overflow-y: auto;
	}
	.links {
		display: flex;
		gap: 0.5rem;
	}
</style>
