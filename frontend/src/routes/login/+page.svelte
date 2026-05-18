<script lang="ts">
	import { onMount } from 'svelte';
	import { login } from '$lib/api/auth';
	import { setAccessToken } from '$lib/api/client';
	import { isAuthenticated } from '$lib/stores/auth';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleLogin(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			await login(email, password);
			$isAuthenticated = true;
			window.location.href = '/';
		} catch (e: any) {
			error = e.message || 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Login - QuickPulse</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-[var(--qp-bg)]">
	<div class="w-full max-w-sm mx-4">
		<div class="text-center mb-8">
			<svg class="w-10 h-10 text-[var(--qp-accent)] mx-auto mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
			</svg>
			<h1 class="text-2xl font-bold text-white">QuickPulse</h1>
			<p class="text-sm text-[var(--qp-text-muted)] mt-1">Sign in to your dashboard</p>
		</div>

		<form onsubmit={handleLogin} class="qp-card space-y-4">
			{#if error}
				<div class="text-sm text-red-400 bg-red-400/10 border border-red-400/20 rounded-lg px-3 py-2">
					{error}
				</div>
			{/if}

			<div>
				<label class="block text-xs text-[var(--qp-text-muted)] mb-1.5">Email</label>
				<input
					type="email"
					class="qp-input"
					bind:value={email}
					placeholder="admin@quickpulse.local"
					required
				/>
			</div>

			<div>
				<label class="block text-xs text-[var(--qp-text-muted)] mb-1.5">Password</label>
				<input
					type="password"
					class="qp-input"
					bind:value={password}
					placeholder="••••••••"
					required
				/>
			</div>

			<button
				type="submit"
				class="qp-btn qp-btn-primary w-full"
				disabled={loading}
			>
				{#if loading}
					<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 12a9 9 0 1 1-6.219-8.56" />
					</svg>
					Signing in...
				{:else}
					Sign in
				{/if}
			</button>
		</form>
	</div>
</div>
