<script lang="ts">
	import { toasts } from '$lib/stores/ui';

	function getToastClass(type: string): string {
		const classes: Record<string, string> = {
			success: 'border-green-500 bg-green-500/10',
			error: 'border-red-500 bg-red-500/10',
			warning: 'border-amber-500 bg-amber-500/10',
			info: 'border-blue-500 bg-blue-500/10',
		};
		return classes[type] || classes.info;
	}
</script>

{#if $toasts.length > 0}
	<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
		{#each $toasts as toast (toast.id)}
			<div
				class="animate-fade-in flex items-center gap-2 px-4 py-3 rounded-lg border {getToastClass(toast.type)} text-sm text-white shadow-lg"
			>
				{toast.message}
			</div>
		{/each}
	</div>
{/if}
