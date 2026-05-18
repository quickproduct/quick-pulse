<script lang="ts">
	import { onMount } from 'svelte';
	import { changePassword } from '$lib/api/auth';
	import { currentUser } from '$lib/stores/auth';
	import { addToast } from '$lib/stores/ui';
	import PageHeader from '$lib/components/layout/PageHeader.svelte';

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let saving = $state(false);

	async function handleChangePassword(e: Event) {
		e.preventDefault();
		if (newPassword !== confirmPassword) {
			addToast('Passwords do not match', 'error');
			return;
		}
		if (newPassword.length < 8) {
			addToast('Password must be at least 8 characters', 'error');
			return;
		}
		saving = true;
		try {
			await changePassword(currentPassword, newPassword);
			addToast('Password changed successfully', 'success');
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
		} catch (e: any) {
			addToast(e.message || 'Failed to change password', 'error');
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Settings - QuickPulse</title>
</svelte:head>

<PageHeader title="Settings" subtitle="Account and application settings" />

<div class="max-w-lg space-y-6">
	<div class="qp-card">
		<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-4">Account</h3>
		{#if $currentUser}
			<div class="space-y-2 text-sm">
				<div class="flex justify-between">
					<span class="text-[var(--qp-text-muted)]">Email</span>
					<span class="text-white">{$currentUser.email}</span>
				</div>
				<div class="flex justify-between">
					<span class="text-[var(--qp-text-muted)]">Role</span>
					<span class="text-white capitalize">{$currentUser.role}</span>
				</div>
			</div>
		{/if}
	</div>

	<div class="qp-card">
		<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-4">Change Password</h3>
		<form onsubmit={handleChangePassword} class="space-y-3">
			<div>
				<label class="block text-xs text-[var(--qp-text-muted)] mb-1">Current Password</label>
				<input type="password" class="qp-input" bind:value={currentPassword} required />
			</div>
			<div>
				<label class="block text-xs text-[var(--qp-text-muted)] mb-1">New Password</label>
				<input type="password" class="qp-input" bind:value={newPassword} required minlength="8" />
			</div>
			<div>
				<label class="block text-xs text-[var(--qp-text-muted)] mb-1">Confirm New Password</label>
				<input type="password" class="qp-input" bind:value={confirmPassword} required />
			</div>
			<button type="submit" class="qp-btn qp-btn-primary w-full" disabled={saving}>
				{saving ? 'Saving...' : 'Change Password'}
			</button>
		</form>
	</div>

	<div class="qp-card">
		<h3 class="text-sm font-medium text-[var(--qp-text-muted)] uppercase tracking-wide mb-4">About</h3>
		<div class="space-y-2 text-sm">
			<div class="flex justify-between">
				<span class="text-[var(--qp-text-muted)]">Application</span>
				<span class="text-white">QuickPulse</span>
			</div>
			<div class="flex justify-between">
				<span class="text-[var(--qp-text-muted)]">Version</span>
				<span class="text-white">0.1.0</span>
			</div>
		</div>
	</div>
</div>
