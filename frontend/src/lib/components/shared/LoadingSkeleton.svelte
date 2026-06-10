<script lang="ts">
	let { rows = 3 } = $props();
</script>

<div class="space-y-3">
	{#each Array(rows) as _, i}
		<div class="qp-card animate-pulse">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-lg bg-[var(--qp-surface-2)]"></div>
				<div class="flex-1 space-y-2">
					<div class="h-3 bg-[var(--qp-surface-2)] rounded w-1/3"></div>
					<div class="h-3 bg-[var(--qp-surface-2)] rounded w-1/2"></div>
				</div>
			</div>
		</div>
	{/each}
</div>
