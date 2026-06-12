<script lang="ts">
	let { message = '' } = $props();
</script>

<div class="flex flex-col items-center justify-center py-16 text-center">
	<svg class="w-12 h-12 text-[var(--qp-text-muted)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
		<path d="M22 12h-4l-3 9L9 3l-3 9H2" />
	</svg>
	<p class="text-[var(--qp-text-muted)] text-sm">{message || 'No data available'}</p>
</div>
