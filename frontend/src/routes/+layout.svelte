<script lang="ts">
	import '../app.css';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { isAuthenticated, initAuth } from '$lib/stores/auth';
	import Sidebar from '$lib/components/layout/Sidebar.svelte';
	import TopNav from '$lib/components/layout/TopNav.svelte';
	import Toast from '$lib/components/shared/Toast.svelte';

	let { children } = $props();
	let loaded = $state(false);

	let isLoginPage = $derived(page.url?.pathname === '/login');
	let isPublicPage = $derived(page.url?.pathname === '/login');

	onMount(async () => {
		await initAuth();
		loaded = true;
	});
</script>

<svelte:head>
	<title>QuickPulse</title>
	<link rel="preconnect" href="https://fonts.googleapis.com">
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous">
	<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
</svelte:head>

{#if !loaded}
	<div class="flex items-center justify-center min-h-screen bg-[var(--qp-bg)]">
		<div class="flex items-center gap-2">
			<svg class="w-6 h-6 text-[var(--qp-accent)] animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M21 12a9 9 0 1 1-6.219-8.56" />
			</svg>
			<span class="text-[var(--qp-text-muted)]">Loading...</span>
		</div>
	</div>
{:else if isPublicPage}
	{@render children()}
{:else if $isAuthenticated}
	<div class="flex min-h-screen">
		<Sidebar />
		<div class="flex-1 flex flex-col min-w-0">
			<TopNav />
			<main class="flex-1 p-6 overflow-y-auto qp-scrollbar">
				{@render children()}
			</main>
		</div>
	</div>
	<Toast />
{:else}
	{#if typeof window !== 'undefined'}
		{goto('/login')}
	{/if}
{/if}
