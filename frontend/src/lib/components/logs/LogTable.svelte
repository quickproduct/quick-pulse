<!--
  LogTable renders a virtual scrolling list of LogRow entries. We avoid
  pulling in a virtual-list dependency to keep the bundle lean — Svelte 5's
  reactivity makes a minimal implementation tractable in ~50 lines.

  Strategy:
    * Fixed row height (28 px) so we can compute the visible window from
      scrollTop alone — no measurement loop, no observers.
    * On scroll, set `first` and update CSS `transform: translateY()` of
      the inner container so only the visible window is rendered.
    * Buffer 20 rows top/bottom so quick scrolls don't flash empty space.
-->
<script lang="ts">
	import type { LogEntry } from '$lib/api/logs';
	import LogRow from './LogRow.svelte';

	let {
		entries,
		selectedId,
		onSelect,
		onReachEnd
	}: {
		entries: LogEntry[];
		selectedId: number | null;
		onSelect: (e: LogEntry) => void;
		onReachEnd?: () => void;
	} = $props();

	const ROW_HEIGHT = 28;
	const OVERSCAN = 20;

	let scrollRef: HTMLDivElement | undefined = $state();
	let scrollTop = $state(0);
	let viewportH = $state(600);

	function onScroll() {
		if (!scrollRef) return;
		scrollTop = scrollRef.scrollTop;
		// Near-bottom detection so the parent can load the next page.
		const distFromBottom =
			scrollRef.scrollHeight - scrollRef.scrollTop - scrollRef.clientHeight;
		if (distFromBottom < 200 && onReachEnd) {
			onReachEnd();
		}
	}

	$effect(() => {
		if (!scrollRef) return;
		const ro = new ResizeObserver(() => {
			viewportH = scrollRef?.clientHeight ?? 600;
		});
		ro.observe(scrollRef);
		viewportH = scrollRef.clientHeight || 600;
		return () => ro.disconnect();
	});

	const first = $derived(Math.max(0, Math.floor(scrollTop / ROW_HEIGHT) - OVERSCAN));
	const visibleCount = $derived(Math.ceil(viewportH / ROW_HEIGHT) + OVERSCAN * 2);
	const last = $derived(Math.min(entries.length, first + visibleCount));
	const offsetY = $derived(first * ROW_HEIGHT);
	const totalH = $derived(entries.length * ROW_HEIGHT);
	const window = $derived(entries.slice(first, last));
</script>

<div class="log-table" bind:this={scrollRef} onscroll={onScroll}>
	{#if entries.length === 0}
		<div class="empty">No logs match the current filter.</div>
	{:else}
		<div class="spacer" style="height: {totalH}px">
			<div class="rows" style="transform: translateY({offsetY}px)">
				{#each window as entry (entry.id)}
					<LogRow
						{entry}
						selected={entry.id === selectedId}
						onClick={onSelect}
					/>
				{/each}
			</div>
		</div>
	{/if}
</div>

<style>
	.log-table {
		flex: 1;
		min-height: 200px;
		overflow-y: auto;
		background: #0d0f18;
		border: 1px solid var(--qp-border);
		border-radius: 0.5rem;
	}
	.log-table::-webkit-scrollbar {
		width: 8px;
	}
	.log-table::-webkit-scrollbar-track {
		background: var(--qp-surface);
	}
	.log-table::-webkit-scrollbar-thumb {
		background: var(--qp-border);
		border-radius: 4px;
	}
	.spacer {
		position: relative;
	}
	.rows {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		will-change: transform;
	}
	.empty {
		padding: 3rem 1rem;
		text-align: center;
		color: var(--qp-text-muted);
		font-size: 0.875rem;
	}
</style>
