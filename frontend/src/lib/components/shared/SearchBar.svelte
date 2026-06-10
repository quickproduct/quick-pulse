<script lang="ts">
	let { value = '', oninput, placeholder = 'Search...' } = $props();
</script>

<div class="relative">
	<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--qp-text-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<circle cx="11" cy="11" r="8" />
		<path d="m21 21-4.35-4.35" />
	</svg>
	<input
		type="text"
		class="qp-input pl-9"
		{placeholder}
		{value}
		oninput={(e) => oninput?.((e.target as HTMLInputElement).value)}
	/>
</div>
