<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { addToast } from '$lib/stores/ui';
	import { getContainerLogs } from '$lib/api/containers';
	import { getAccessToken } from '$lib/api/client';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';

	let id: string = $derived(page.params.id || '');
	let logs: string[] = $state([]);
	let paused = $state(false);
	let tail = $state(100);
	let loading = $state(true);
	let autoScroll = $state(true);
	let logContainer: HTMLDivElement | undefined = $state();
	let ws: WebSocket | undefined;
	let wsConnected = $state(false);

	async function loadInitialLogs() {
		loading = true;
		try {
			const result = await getContainerLogs(id, tail);
			logs = result?.logs || [];
		} catch (e: any) {
			addToast(e.message || 'Failed to load logs', 'error');
		} finally {
			loading = false;
		}
	}

	function connectWS() {
		const token = getAccessToken();
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const host = window.location.host;
		ws = new WebSocket(`${protocol}//${host}/ws/logs/${id}?token=${token}`);

		ws.onopen = () => {
			wsConnected = true;
		};

		ws.onmessage = (event) => {
			if (paused) return;
			try {
				const data = JSON.parse(event.data);
				if (data.line) {
					logs = [...logs, data.line];
					if (logs.length > 2000) {
						logs = logs.slice(-1000);
					}
					if (autoScroll && logContainer) {
						const raf = requestAnimationFrame(() => {
							if (logContainer) logContainer.scrollTop = logContainer.scrollHeight;
						});
					}
				}
			} catch (e) {
				console.warn('[ws/logs] parse error:', e);
			}
		};

		ws.onerror = (event) => {
			console.error('[ws/logs] error:', event);
			wsConnected = false;
		};

		ws.onclose = (event) => {
			wsConnected = false;
			if (event.code !== 1000) {
				setTimeout(connectWS, 3000);
			}
		};
	}

	function togglePause() {
		paused = !paused;
		if (ws && ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ action: paused ? 'pause' : 'resume' }));
		}
	}

	onMount(() => {
		loadInitialLogs();
		connectWS();
	});

	onDestroy(() => {
		if (ws) {
			ws.onclose = null;
			ws.close(1000);
		}
	});
</script>

<svelte:head>
	<title>Logs - {id} - QuickPulse</title>
</svelte:head>

<div class="flex items-center justify-between mb-4 flex-wrap gap-3">
	<div>
		<a href="/containers/{id}" class="text-xs text-[var(--qp-accent)] hover:underline">&larr; Back to container</a>
		<h1 class="text-xl font-semibold text-white mt-1">Live Logs</h1>
		<p class="text-sm text-[var(--qp-text-muted)]">{id}</p>
	</div>
	<div class="flex items-center gap-3">
		{#if wsConnected}
			<span class="flex items-center gap-1.5 text-xs text-green-400">
				<span class="w-1.5 h-1.5 rounded-full bg-green-400 pulse-dot"></span>
				Live
			</span>
		{:else}
			<span class="flex items-center gap-1.5 text-xs text-amber-400">
				<span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
				Connecting...
			</span>
		{/if}
		<button class="qp-btn qp-btn-ghost text-xs" onclick={togglePause}>
			{#if paused}
				<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3" /></svg>
				Resume
			{:else}
				<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" /><rect x="14" y="4" width="4" height="16" /></svg>
				Pause
			{/if}
		</button>
		<label class="flex items-center gap-1 text-xs text-[var(--qp-text-muted)] cursor-pointer">
			<input type="checkbox" bind:checked={autoScroll} class="rounded" />
			Auto-scroll
		</label>
	</div>
</div>

<div class="log-viewer qp-scrollbar" bind:this={logContainer}>
	{#if loading}
		<div class="text-[var(--qp-text-muted)]">Loading logs...</div>
	{:else if logs.length === 0}
		<div class="text-[var(--qp-text-muted)]">No logs available</div>
	{:else}
		{#each logs as line, i}
			<div class="log-line"><span class="text-[var(--qp-text-muted)] select-none mr-2">{String(i + 1).padStart(4)}</span>{line}</div>
		{/each}
	{/if}
</div>
