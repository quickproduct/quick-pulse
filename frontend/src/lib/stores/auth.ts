import { writable } from 'svelte/store';
import { getAccessToken } from '../api/client';
import type { UserResponse } from '../api/auth';
import { getMe } from '../api/auth';

export const isAuthenticated = writable(false);
export const currentUser = writable<UserResponse | null>(null);

export async function initAuth() {
	const token = getAccessToken();
	if (token) {
		try {
			const user = await getMe();
			currentUser.set(user);
			isAuthenticated.set(true);
		} catch {
			isAuthenticated.set(false);
			currentUser.set(null);
		}
	} else {
		isAuthenticated.set(false);
		currentUser.set(null);
	}
}
