<!--
  Centralized logs page. Composes:
    * LogFilters    — search box + level/platform/source filters
    * LogTable      — virtual-scrolling table of LogRow entries
    * LogDetailDrawer — slide-in panel for one entry
    * Live tail     — toggled by a button; connects /ws/logs/stream

  State design:
    * `filter`  drives every network call. URL search params mirror it so
      views are shareable and back/forward works.
    * `entries` holds the visible window. We prepend incoming live entries
      (dedup by id) and append historical pages from cursor-based load-more.
    * Memory cap: 5000 entries in-memory. When live tail crosses that
      threshold we drop the oldest 500 — same approach the existing per-pod
      log viewer uses.
-->
<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		searchLogs,
		getSources,
		exportUrl,
		type LogEntry,
		type LogsFilter,
		type LogsSources
	} from '$lib/api/logs';
	import { getAccessToken } from '$lib/api/client';
	import { addToast } from '$lib/stores/ui';
	import LogFilters from '$lib/components/logs/LogFilters.svelte';
	import LogTable from '$lib/components/logs/LogTable.svelte';
	import LogDetailDrawer from '$lib/components/logs/LogDetailDrawer.svelte';

	const MAX_ENTRIES = 5000;
	const TRIM_TO = 4500;
	const PAGE_SIZE = 200;

	let filter: LogsFilter = $state(filterFromURL());
	let entries: LogEntry[] = $state([]);
	let cursor = $state(''); // for "load more" pagination
	let loading = $state(false);
	let sources: LogsSources | null = $state(null);
	let selected: LogEntry | null = $state(null);
	let liveOn = $state(true);
	let paused = $state(false);
	let wsState = $state<'connecting' | 'open' | 'closed' | 'error'>('closed');
	let droppedNotice = $state(0);
	let ws: WebSocket | undefined;
	let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
	let searchDebounce: ReturnType<typeof setTimeout> | undefined;

	function filterFromURL(): LogsFilter {
		if (typeof window === 'undefined') return { limit: PAGE_SIZE };
		const sp = new URLSearchParams(window.location.search);
		const list = (k: string) => {
			const v = sp.get(k);
			return v ? v.split(',') : undefined;
		};
		return {
			q: sp.get('q') ?? undefined,
			level: list('level'),
			platform: list('platform'),
			cluster: list('cluster'),
			container: list('container'),
			pod: list('pod'),
			namespace: list('namespace'),
			service: list('service'),
			env: list('env'),
			limit: PAGE_SIZE
		};
	}

	function persistFilterToURL(f: LogsFilter) {
		const sp = new URLSearchParams();
		if (f.q) sp.set('q', f.q);
		for (const k of ['level', 'platform', 'cluster', 'container', 'pod', 'namespace', 'service', 'env'] as const) {
			const v = f[k] as string[] | undefined;
			if (v && v.length) sp.set(k, v.join(','));
		}
		const qs = sp.toString();
		const target = qs ? `/logs?${qs}` : '/logs';
		// Use history.replaceState so we don't pollute history on every keystroke.
		history.replaceState(null, '', target);
	}

	async function reload() {
		loading = true;
		try {
			const res = await searchLogs({ ...filter, cursor: '', limit: PAGE_SIZE });
			entries = res.logs;
			cursor = res.next_cursor ?? '';
		} catch (e: any) {
			addToast(e.message || 'Failed to load logs', 'error');
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		if (!cursor || loading) return;
		loading = true;
		try {
			const res = await searchLogs({ ...filter, cursor, limit: PAGE_SIZE });
			// Append (results are newest-first; cursor pages older).
			const newIds = new Set(entries.map((e) => e.id));
			const fresh = res.logs.filter((e) => !newIds.has(e.id));
			entries = [...entries, ...fresh];
			cursor = res.next_cursor ?? '';
		} catch (e: any) {
			addToast(e.message || 'Failed to load more', 'error');
		} finally {
			loading = false;
		}
	}

	function onFilterChange(next: LogsFilter) {
		filter = next;
		persistFilterToURL(next);
		// Debounce reloads so typing in the search box isn't a network storm.
		if (searchDebounce) clearTimeout(searchDebounce);
		searchDebounce = setTimeout(() => {
			reload();
			updateLiveFilter();
		}, 250);
	}

	// --- Live tail ----------------------------------------------------------

	function buildLiveFilter() {
		// The broker accepts the same shape; we just send what's set.
		return {
			level: filter.level,
			platform: filter.platform,
			cluster: filter.cluster,
			container: filter.container,
			pod: filter.pod,
			namespace: filter.namespace,
			service: filter.service,
			env: filter.env,
			q: filter.q
		};
	}

	function connectWS() {
		if (!liveOn) return;
		closeWS();
		wsState = 'connecting';
		const token = getAccessToken();
		const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		ws = new WebSocket(`${proto}//${window.location.host}/ws/logs/stream?token=${token}`);
		ws.onopen = () => {
			wsState = 'open';
			ws?.send(JSON.stringify({ action: 'subscribe', filter: buildLiveFilter() }));
		};
		ws.onmessage = (ev) => {
			if (paused) return;
			try {
				const msg = JSON.parse(ev.data);
				if (msg.logs && Array.isArray(msg.logs)) {
					prependEntries(msg.logs as LogEntry[]);
				} else if (typeof msg.dropped === 'number') {
					droppedNotice += msg.dropped;
				} else if (msg.error) {
					addToast(msg.error, 'error');
				}
			} catch {}
		};
		ws.onerror = () => {
			wsState = 'error';
		};
		ws.onclose = (ev) => {
			wsState = 'closed';
			if (liveOn && ev.code !== 1000) {
				reconnectTimer = setTimeout(connectWS, 3000);
			}
		};
	}

	function closeWS() {
		if (reconnectTimer) {
			clearTimeout(reconnectTimer);
			reconnectTimer = undefined;
		}
		if (ws) {
			ws.onclose = null;
			ws.close(1000);
			ws = undefined;
		}
	}

	function prependEntries(fresh: LogEntry[]) {
		if (fresh.length === 0) return;
		const seen = new Set(entries.map((e) => e.id));
		const additions = fresh.filter((e) => !seen.has(e.id));
		if (additions.length === 0) return;
		// Newest first within the batch.
		additions.sort((a, b) => b.ts - a.ts || b.id - a.id);
		entries = [...additions, ...entries];
		if (entries.length > MAX_ENTRIES) {
			entries = entries.slice(0, TRIM_TO);
			// Once we've trimmed live entries, the "load more" cursor is
			// no longer meaningful — clear it so the user reloads instead.
			cursor = '';
		}
	}

	function updateLiveFilter() {
		if (ws?.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ action: 'filter', filter: buildLiveFilter() }));
		}
	}

	function toggleLive() {
		liveOn = !liveOn;
		if (liveOn) {
			connectWS();
		} else {
			closeWS();
		}
	}

	function togglePause() {
		paused = !paused;
		if (ws?.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ action: paused ? 'pause' : 'resume' }));
		}
	}

	// --- Lifecycle ----------------------------------------------------------

	onMount(() => {
		async function init() {
			// Sources are best-effort; the filter UI still works without them.
			try {
				sources = await getSources();
			} catch {}
			await reload();
		}
		init();
		connectWS();
		// Refresh sources occasionally so newly-created containers appear
		// in the dropdowns. 30s matches the docker discovery cadence.
		const sIv = setInterval(async () => {
			try {
				sources = await getSources();
			} catch {}
		}, 30_000);
		return () => clearInterval(sIv);
	});

	onDestroy(() => {
		closeWS();
		if (searchDebounce) clearTimeout(searchDebounce);
	});

	function exportLink(fmt: 'csv' | 'json') {
		return exportUrl(filter, fmt);
	}

	const wsLabel = $derived(
		wsState === 'open' && !paused ? 'Live'
		: wsState === 'open' && paused ? 'Paused'
		: wsState === 'connecting' ? 'Connecting…'
		: wsState === 'error' ? 'Error'
		: 'Disconnected'
	);
	const wsDot = $derived(
		wsState === 'open' && !paused ? 'bg-emerald-400'
		: wsState === 'open' && paused ? 'bg-amber-400'
		: wsState === 'error' ? 'bg-red-400'
		: 'bg-slate-400'
	);
</script>

<svelte:head>
	<title>Logs - QuickPulse</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<div>
			<h1 class="title">Logs</h1>
			<p class="subtitle">Centralized logs from Docker containers and Kubernetes pods.</p>
		</div>
		<div class="header-actions">
			{#if sources}
				<span class="stat" title="Active streamers">
					streams: <strong>{sources.active_streams}</strong>
				</span>
				{#if sources.dropped > 0}
					<span class="stat-warn" title="Total entries dropped under backpressure since startup">
						dropped: <strong>{sources.dropped}</strong>
					</span>
				{/if}
			{/if}
			<div class="live-pill">
				<span class="dot {wsDot} {wsState === 'open' && !paused ? 'pulse-dot' : ''}"></span>
				{wsLabel}
			</div>
			<button class="qp-btn qp-btn-ghost xs" onclick={toggleLive}>{liveOn ? 'Stop live' : 'Start live'}</button>
			<button class="qp-btn qp-btn-ghost xs" onclick={togglePause} disabled={!liveOn || wsState !== 'open'}>
				{paused ? 'Resume' : 'Pause'}
			</button>
			<a class="qp-btn qp-btn-ghost xs" href={exportLink('csv')}>Export CSV</a>
			<a class="qp-btn qp-btn-ghost xs" href={exportLink('json')}>Export JSON</a>
		</div>
	</header>

	{#if droppedNotice > 0}
		<div class="dropped-banner">
			⚠ {droppedNotice} live log line(s) dropped — your filter is producing too much volume.
			<button class="qp-btn-ghost dismiss" onclick={() => (droppedNotice = 0)}>Dismiss</button>
		</div>
	{/if}

	<LogFilters {filter} {sources} onChange={onFilterChange} />

	<div class="table-wrap">
		<LogTable
			{entries}
			selectedId={selected?.id ?? null}
			onSelect={(e) => (selected = e)}
			onReachEnd={loadMore}
		/>
		{#if loading}
			<div class="loading-overlay">Loading…</div>
		{/if}
	</div>

	<LogDetailDrawer entry={selected} onClose={() => (selected = null)} />
</div>

<style>
	.page {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		padding: 1.5rem;
		height: 100vh;
		max-height: 100vh;
	}
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 1rem;
		flex-wrap: wrap;
	}
	.title {
		font-size: 1.5rem;
		font-weight: 700;
		color: var(--qp-text);
		margin: 0;
	}
	.subtitle {
		color: var(--qp-text-muted);
		font-size: 0.875rem;
		margin: 0.25rem 0 0 0;
	}
	.header-actions {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		flex-wrap: wrap;
	}
	.stat,
	.stat-warn {
		font-size: 0.75rem;
		color: var(--qp-text-muted);
		padding: 0.25rem 0.5rem;
		background: var(--qp-surface-2);
		border-radius: 0.375rem;
	}
	.stat-warn {
		color: #f59e0b;
		background: rgba(245, 158, 11, 0.1);
	}
	.live-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.25rem 0.625rem;
		border-radius: 9999px;
		background: var(--qp-surface);
		border: 1px solid var(--qp-border);
		font-size: 0.75rem;
		color: var(--qp-text-muted);
		font-family: var(--font-family-mono);
	}
	.dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		display: inline-block;
	}
	.bg-emerald-400 { background-color: #34d399; }
	.bg-amber-400 { background-color: #fbbf24; }
	.bg-red-400 { background-color: #f87171; }
	.bg-slate-400 { background-color: #94a3b8; }
	.xs {
		padding: 0.3rem 0.6rem;
		font-size: 0.75rem;
	}
	.dropped-banner {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.5rem 0.75rem;
		background: rgba(245, 158, 11, 0.15);
		border: 1px solid rgba(245, 158, 11, 0.4);
		color: #fbbf24;
		border-radius: 0.5rem;
		font-size: 0.8125rem;
	}
	.dismiss {
		background: transparent;
		border: none;
		color: #fbbf24;
		cursor: pointer;
		text-decoration: underline;
		font-size: 0.75rem;
	}
	.table-wrap {
		flex: 1;
		display: flex;
		flex-direction: column;
		position: relative;
		min-height: 0;
	}
	.loading-overlay {
		position: absolute;
		bottom: 0.75rem;
		right: 0.75rem;
		background: var(--qp-surface-2);
		color: var(--qp-text-muted);
		font-size: 0.75rem;
		padding: 0.25rem 0.5rem;
		border-radius: 0.375rem;
		pointer-events: none;
	}
</style>
