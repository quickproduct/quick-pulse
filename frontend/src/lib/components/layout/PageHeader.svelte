<script lang="ts">
	let { title = '', subtitle = '' } = $props();
</script>

<div class="mb-6">
	{#if title}
		<h1 class="text-xl font-semibold text-white">{title}</h1>
	{/if}
	{#if subtitle}
		<p class="text-sm text-[var(--qp-text-muted)] mt-1">{subtitle}</p>
	{/if}
</div>
