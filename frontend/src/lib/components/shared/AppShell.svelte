<script lang="ts">
	let { children } = $props();
</script>

<div class="flex min-h-screen bg-[var(--qp-bg)]">
	{@render children()}
</div>
