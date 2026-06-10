<script lang="ts">
	import { page } from '$app/state';

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: 'grid' },
		{ href: '/containers', label: 'Containers', icon: 'box' },
		{ href: '/stacks', label: 'Stacks', icon: 'layers' },
		{ href: '/kubernetes', label: 'Kubernetes', icon: 'k8s' },
		{ href: '/logs', label: 'Logs', icon: 'logs' },
		{ href: '/events', label: 'Events', icon: 'zap' },
		{ href: '/alerts', label: 'Alerts', icon: 'bell' },
		{ href: '/settings', label: 'Settings', icon: 'settings' },
	];

	function getIcon(icon: string): string {
		const icons: Record<string, string> = {
			grid: 'M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z',
			box: 'M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z',
			layers: 'M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5',
			k8s: 'M12 2l9 4.5v11L12 22l-9-4.5v-11L12 2z M12 6l6 3v6l-6 3l-6-3V9l6-3',
			zap: 'M13 2L3 14h9l-1 8 10-12h-9l1-8z',
			logs: 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6 M16 13H8 M16 17H8 M10 9H8',
			bell: 'M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 0 1-3.46 0',
			settings: 'M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
			book: 'M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253',
		};
		return icons[icon] || '';
	}

	let currentPath = $derived(page.url?.pathname || '/');
	let collapsed = $state(false);

	function isActive(href: string): boolean {
		if (href === '/') {
			return currentPath === '/';
		}
		return currentPath.startsWith(href);
	}
</script>

<aside
	class="flex flex-col shrink-0 border-r border-[var(--qp-border)] bg-[var(--qp-surface)] transition-all duration-200 {collapsed ? 'w-16' : 'w-56'}"
>
	<div class="flex items-center gap-2 px-4 py-5 border-b border-[var(--qp-border)]">
		{#if !collapsed}
			<div class="flex items-center gap-2">
				<svg class="w-6 h-6 text-[var(--qp-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
				</svg>
				<span class="text-lg font-bold text-white">QuickPulse</span>
			</div>
		{:else}
			<svg class="w-6 h-6 text-[var(--qp-accent)] mx-auto" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
			</svg>
		{/if}
	</div>

	<nav class="flex-1 py-2">
		{#each navItems as item}
			<a
				href={item.href}
				class="flex items-center gap-3 px-4 py-2.5 mx-2 rounded-lg text-sm transition-colors {isActive(item.href)
					? 'bg-[var(--qp-accent)] text-white'
					: 'text-[var(--qp-text-muted)] hover:bg-[var(--qp-surface-2)] hover:text-[var(--qp-text)]'}"
			>
				<svg class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d={getIcon(item.icon)} />
				</svg>
				{#if !collapsed}
					<span>{item.label}</span>
				{/if}
			</a>
		{/each}
	</nav>

	<button
		onclick={() => collapsed = !collapsed}
		class="flex items-center justify-center py-3 border-t border-[var(--qp-border)] text-[var(--qp-text-muted)] hover:text-[var(--qp-text)] transition-colors"
	>
		<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			{#if collapsed}
				<path d="M9 18l6-6-6-6" />
			{:else}
				<path d="M15 18l-6-6 6-6" />
			{/if}
		</svg>
	</button>
</aside>
