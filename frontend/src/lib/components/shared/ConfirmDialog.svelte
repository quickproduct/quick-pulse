<script lang="ts">
	let {
		open = false,
		title = 'Confirm',
		message = '',
		confirmLabel = 'Confirm',
		confirmClass = 'qp-btn-danger',
		onconfirm,
		oncancel,
	} = $props();
</script>

{#if open}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onclick={oncancel}>
		<div class="qp-card w-full max-w-md mx-4" onclick={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-semibold text-white mb-2">{title}</h3>
			<p class="text-sm text-[var(--qp-text-muted)] mb-6">{message}</p>
			<div class="flex justify-end gap-3">
				<button class="qp-btn qp-btn-ghost" onclick={oncancel}>Cancel</button>
				<button class="qp-btn {confirmClass}" onclick={onconfirm}>{confirmLabel}</button>
			</div>
		</div>
	</div>
{/if}
