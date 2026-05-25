<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { getAccessToken } from '$lib/api/client';
	import { addToast } from '$lib/stores/ui';

	let id: string = $derived(page.params.id || '');
	let terminalContainer: HTMLDivElement | undefined = $state();
	let ws: WebSocket | undefined;
	let term: any;
	let fitAddon: any;
	let wsConnected = $state(false);

	async function initTerminal() {
		// Import xterm and fit-addon dynamically to prevent SSR failures
		const { Terminal } = await import('xterm');
		const { FitAddon } = await import('xterm-addon-fit');

		term = new Terminal({
			cursorBlink: true,
			theme: {
				background: '#0d0e12',
				foreground: '#f3f4f6',
				cursor: '#3b82f6',
				black: '#000000',
				red: '#ef4444',
				green: '#10b981',
				yellow: '#f59e0b',
				blue: '#3b82f6',
				magenta: '#8b5cf6',
				cyan: '#06b6d4',
				white: '#ffffff'
			},
			fontFamily: 'JetBrains Mono, Menlo, Monaco, Consolas, monospace',
			fontSize: 13,
			lineHeight: 1.2
		});

		fitAddon = new FitAddon();
		term.loadAddon(fitAddon);

		if (terminalContainer) {
			term.open(terminalContainer);
			fitAddon.fit();
		}

		term.onData((data: string) => {
			if (ws && ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ action: 'input', data }));
			}
		});

		window.addEventListener('resize', handleResize);
	}

	function sendResize() {
		if (term && ws && ws.readyState === WebSocket.OPEN) {
			const cols = term.cols;
			const rows = term.rows;
			ws.send(JSON.stringify({ action: 'resize', cols, rows }));
		}
	}

	function handleResize() {
		if (fitAddon) {
			fitAddon.fit();
			sendResize();
		}
	}

	function connectWS() {
		const token = getAccessToken();
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const host = window.location.host;
		ws = new WebSocket(`${protocol}//${host}/ws/containers/${id}/terminal?token=${token}`);

		ws.onopen = () => {
			wsConnected = true;
			if (term) {
				term.write('\r\n\x1b[1;32mConnected to container shell.\x1b[0m\r\n');
				sendResize();
			}
		};

		ws.onmessage = (event) => {
			if (term) {
				term.write(event.data);
			}
		};

		ws.onerror = (event) => {
			console.error('[ws/terminal] error:', event);
			wsConnected = false;
		};

		ws.onclose = (event) => {
			wsConnected = false;
			if (term) {
				term.write('\r\n\x1b[1;31mConnection closed.\x1b[0m\r\n');
			}
		};
	}

	onMount(async () => {
		await initTerminal();
		connectWS();
	});

	onDestroy(() => {
		window.removeEventListener('resize', handleResize);
		if (ws) {
			ws.onclose = null;
			ws.close(1000);
		}
		if (term) {
			term.dispose();
		}
	});
</script>

<svelte:head>
	<title>Terminal - {id} - QuickPulse</title>
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.3.0/css/xterm.min.css" />
</svelte:head>

<div class="flex items-center justify-between mb-4 flex-wrap gap-3">
	<div>
		<a href="/containers/{id}" class="text-xs text-[var(--qp-accent)] hover:underline">&larr; Back to container</a>
		<h1 class="text-xl font-semibold text-white mt-1">Interactive Terminal</h1>
		<p class="text-sm text-[var(--qp-text-muted)]">{id}</p>
	</div>
	<div>
		{#if wsConnected}
			<span class="flex items-center gap-1.5 text-xs text-green-400">
				<span class="w-1.5 h-1.5 rounded-full bg-green-400 pulse-dot"></span>
				Connected
			</span>
		{:else}
			<span class="flex items-center gap-1.5 text-xs text-amber-400">
				<span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
				Connecting...
			</span>
		{/if}
	</div>
</div>

<div class="qp-card p-2 bg-[#0d0e12] border border-[var(--qp-border)] rounded-lg">
	<div bind:this={terminalContainer} class="h-[65vh] w-full overflow-hidden"></div>
</div>

<style>
	:global(.xterm) {
		padding: 10px;
		height: 100%;
	}
</style>
