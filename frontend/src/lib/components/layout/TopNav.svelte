<script lang="ts">
	import { currentUser, isAuthenticated } from '$lib/stores/auth';
	import { logout } from '$lib/api/auth';
	import { page } from '$app/state';

	async function handleLogout() {
		await logout();
		$isAuthenticated = false;
		$currentUser = null;
		window.location.href = '/login';
	}

	let pageTitle = $derived(
		(page.url?.pathname as string) === '/' || (page.url?.pathname as string) === '/index.html'
			? 'Dashboard'
			: page.url?.pathname?.split('/')[1]?.charAt(0).toUpperCase() + page.url?.pathname?.split('/')[1]?.slice(1) || 'Console'
	);
</script>

<header class="flex items-center justify-between px-6 py-3 border-b border-[var(--qp-border)] bg-[var(--qp-surface)]">
	<div class="flex items-center gap-2">
		<span class="text-sm font-semibold text-white tracking-wide">{pageTitle}</span>
	</div>
	<div class="flex items-center gap-4">
		{#if $isAuthenticated && $currentUser}
			<span class="text-sm text-[var(--qp-text-muted)]">{$currentUser.email}</span>
			<button onclick={handleLogout} class="qp-btn qp-btn-ghost text-xs">Logout</button>
		{/if}
	</div>
</header>
