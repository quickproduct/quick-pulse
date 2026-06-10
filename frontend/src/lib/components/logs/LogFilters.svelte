<!--
  LogFilters: a single horizontal filter bar holding the FTS query, level
  multi-select, platform toggle, and source pickers. Emits an updated
  LogsFilter via `onChange` whenever the user changes anything; the parent
  debounces the network call.
-->
<script lang="ts">
	import type { LogsFilter, LogsSources } from '$lib/api/logs';

	let {
		filter,
		sources,
		onChange
	}: {
		filter: LogsFilter;
		sources: LogsSources | null;
		onChange: (next: LogsFilter) => void;
	} = $props();

	const LEVELS = ['DEBUG', 'INFO', 'WARN', 'ERROR', 'CRITICAL'];

	function toggleLevel(lv: string) {
		const cur = new Set(filter.level ?? []);
		if (cur.has(lv)) cur.delete(lv);
		else cur.add(lv);
		onChange({ ...filter, level: [...cur] });
	}

	function togglePlatform(p: string) {
		const cur = new Set(filter.platform ?? []);
		if (cur.has(p)) cur.delete(p);
		else cur.add(p);
		onChange({ ...filter, platform: [...cur] });
	}

	function selectFrom(field: keyof LogsFilter, value: string) {
		if (!value) return;
		const cur = new Set((filter[field] as string[]) ?? []);
		cur.add(value);
		onChange({ ...filter, [field]: [...cur] });
	}

	function removeFrom(field: keyof LogsFilter, value: string) {
		const cur = ((filter[field] as string[]) ?? []).filter((v) => v !== value);
		onChange({ ...filter, [field]: cur });
	}

	function clearAll() {
		onChange({ q: '', limit: filter.limit });
	}

	// Local controlled value for the search box.
	let q = $derived(filter.q ?? '');

	// Time range options
	let selectedTimeRange = $derived.by(() => {
		if (!filter.from) return 'all';
		const diff = Date.now() - filter.from;
		if (diff <= 16 * 60 * 1000) return '15m';
		if (diff <= 61 * 60 * 1000) return '1h';
		if (diff <= 4.1 * 60 * 60 * 1000) return '4h';
		if (diff <= 24.1 * 60 * 60 * 1000) return '24h';
		return 'custom';
	});

	function handleTimeRangeChange(val: string) {
		let from: number | undefined = undefined;
		const now = Date.now();
		if (val === '15m') {
			from = now - 15 * 60 * 1000;
		} else if (val === '1h') {
			from = now - 60 * 60 * 1000;
		} else if (val === '4h') {
			from = now - 4 * 60 * 60 * 1000;
		} else if (val === '24h') {
			from = now - 24 * 60 * 60 * 1000;
		}
		onChange({ ...filter, from });
	}
</script>

<div class="filter-card">
	<!-- Top Row: Search & Time Range -->
	<div class="top-row">
		<div class="search-wrapper">
			<svg class="search-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<input
				type="text"
				class="qp-input search-input"
				placeholder="Search logs (full-text)…"
				value={q}
				oninput={(e) => onChange({ ...filter, q: (e.currentTarget as HTMLInputElement).value })}
			/>
		</div>

		<!-- Time range dropdown -->
		<div class="time-wrapper">
			<select
				class="qp-input select-time font-mono"
				value={selectedTimeRange}
				onchange={(e) => handleTimeRangeChange(e.currentTarget.value)}
				title="Time Range"
			>
				<option value="all">🕒 All Time</option>
				<option value="15m">🕒 Last 15 mins</option>
				<option value="1h">🕒 Last 1 hour</option>
				<option value="4h">🕒 Last 4 hours</option>
				<option value="24h">🕒 Last 24 hours</option>
				{#if selectedTimeRange === 'custom'}
					<option value="custom">🕒 Custom Range</option>
				{/if}
			</select>
		</div>

		<button class="qp-btn qp-btn-ghost clear-btn" onclick={clearAll}>
			Clear
		</button>
	</div>

	<!-- Bottom Row: Multi-select levels and platform, followed by dropdown pickers -->
	<div class="filters-grid">
		<!-- Levels -->
		<div class="filter-group">
			<span class="group-label">Level</span>
			<div class="group-content">
				{#each LEVELS as lv}
					{@const active = (filter.level ?? []).includes(lv)}
					<button
						class="chip {active ? 'chip-active' : ''}"
						onclick={() => toggleLevel(lv)}
					>
						{lv}
					</button>
				{/each}
			</div>
		</div>

		<!-- Platform -->
		<div class="filter-group">
			<span class="group-label">Platform</span>
			<div class="group-content">
				{#each ['docker', 'k8s'] as p}
					{@const active = (filter.platform ?? []).includes(p)}
					<button class="chip {active ? 'chip-active' : ''}" onclick={() => togglePlatform(p)}>
						{p === 'k8s' ? '☸ k8s' : '🐳 docker'}
					</button>
				{/each}
			</div>
		</div>

		<!-- Source Pickers -->
		{#if sources}
			<div class="pickers-group">
				{#if sources.clusters?.length > 1}
					<select
						class="qp-input picker"
						onchange={(e) => {
							selectFrom('cluster', (e.currentTarget as HTMLSelectElement).value);
							(e.currentTarget as HTMLSelectElement).value = '';
						}}
					>
						<option value="">+ Cluster</option>
						{#each sources.clusters as cl}<option value={cl}>{cl}</option>{/each}
					</select>
				{/if}
				<select
					class="qp-input picker"
					onchange={(e) => {
						selectFrom('container', (e.currentTarget as HTMLSelectElement).value);
						(e.currentTarget as HTMLSelectElement).value = '';
					}}
				>
					<option value="">+ Container</option>
					{#each sources.containers as c}<option value={c}>{c}</option>{/each}
				</select>
				<select
					class="qp-input picker"
					onchange={(e) => {
						selectFrom('namespace', (e.currentTarget as HTMLSelectElement).value);
						(e.currentTarget as HTMLSelectElement).value = '';
					}}
				>
					<option value="">+ Namespace</option>
					{#each sources.namespaces as ns}<option value={ns}>{ns}</option>{/each}
				</select>
				<select
					class="qp-input picker"
					onchange={(e) => {
						selectFrom('pod', (e.currentTarget as HTMLSelectElement).value);
						(e.currentTarget as HTMLSelectElement).value = '';
					}}
				>
					<option value="">+ Pod</option>
					{#each sources.pods as p}<option value={p}>{p}</option>{/each}
				</select>
				<select
					class="qp-input picker"
					onchange={(e) => {
						selectFrom('service', (e.currentTarget as HTMLSelectElement).value);
						(e.currentTarget as HTMLSelectElement).value = '';
					}}
				>
					<option value="">+ Service</option>
					{#each sources.services as s}<option value={s}>{s}</option>{/each}
				</select>
			</div>
		{/if}
	</div>

	<!-- Selected/Active Filters Row -->
	{#if (filter.cluster?.length || filter.container?.length || filter.pod?.length || filter.namespace?.length || filter.service?.length)}
		<div class="selected-row font-mono">
			{#each filter.cluster ?? [] as v}
				<button class="chip chip-removable" onclick={() => removeFrom('cluster', v)}>cluster:{v} ✕</button>
			{/each}
			{#each filter.container ?? [] as v}
				<button class="chip chip-removable" onclick={() => removeFrom('container', v)}>container:{v} ✕</button>
			{/each}
			{#each filter.namespace ?? [] as v}
				<button class="chip chip-removable" onclick={() => removeFrom('namespace', v)}>ns:{v} ✕</button>
			{/each}
			{#each filter.pod ?? [] as v}
				<button class="chip chip-removable" onclick={() => removeFrom('pod', v)}>pod:{v} ✕</button>
			{/each}
			{#each filter.service ?? [] as v}
				<button class="chip chip-removable" onclick={() => removeFrom('service', v)}>svc:{v} ✕</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.filter-card {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding: 1rem;
		background: var(--qp-surface);
		border: 1px solid var(--qp-border);
		border-radius: 0.75rem;
		box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
	}
	.top-row {
		display: flex;
		gap: 0.75rem;
		align-items: center;
	}
	.search-wrapper {
		position: relative;
		flex: 1;
	}
	.search-icon {
		position: absolute;
		left: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		width: 1rem;
		height: 1rem;
		color: var(--qp-text-muted);
		pointer-events: none;
	}
	.search-input {
		padding-left: 2.25rem;
	}
	.time-wrapper {
		width: 180px;
		flex-shrink: 0;
	}
	.select-time {
		appearance: none;
		background-image: url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e");
		background-repeat: no-repeat;
		background-position: right 0.75rem center;
		background-size: 1em;
		padding-right: 2.25rem;
	}
	.clear-btn {
		flex-shrink: 0;
		padding: 0.5rem 1rem;
	}
	.filters-grid {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
		align-items: center;
		padding-top: 0.25rem;
	}
	.filter-group {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid var(--qp-border);
		padding: 0.25rem 0.5rem;
		border-radius: 0.5rem;
	}
	.group-label {
		font-size: 0.65rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--qp-text-muted);
		border-right: 1px solid var(--qp-border);
		padding-right: 0.5rem;
	}
	.group-content {
		display: flex;
		gap: 0.25rem;
		align-items: center;
	}
	.pickers-group {
		display: flex;
		gap: 0.375rem;
		flex-wrap: wrap;
	}
	.chip {
		font-family: var(--font-family-mono);
		font-size: 0.7rem;
		padding: 0.15rem 0.5rem;
		border-radius: 0.375rem;
		background: transparent;
		color: var(--qp-text-muted);
		border: 1px solid transparent;
		cursor: pointer;
		transition: all 0.15s ease;
	}
	.chip:hover {
		color: var(--qp-text);
		background: rgba(255, 255, 255, 0.04);
	}
	.chip-active {
		background: rgba(99, 102, 241, 0.12);
		color: var(--qp-accent-hover);
		border-color: rgba(99, 102, 241, 0.3);
	}
	.chip-removable {
		background: rgba(99, 102, 241, 0.06);
		color: var(--qp-accent-hover);
		border: 1px solid rgba(99, 102, 241, 0.2);
	}
	.chip-removable:hover {
		background: rgba(239, 68, 68, 0.1);
		color: #ef4444;
		border-color: rgba(239, 68, 68, 0.2);
	}
	.picker {
		width: auto;
		min-width: 120px;
		height: 32px;
		padding: 0 1.75rem 0 0.5rem;
		font-size: 0.75rem;
		appearance: none;
		background-image: url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e");
		background-repeat: no-repeat;
		background-position: right 0.5rem center;
		background-size: 0.85em;
	}
	.selected-row {
		display: flex;
		gap: 0.375rem;
		flex-wrap: wrap;
		padding-top: 0.5rem;
		border-top: 1px dashed var(--qp-border);
	}
</style>
