import { apiFetch, setAccessToken, clearAccessToken } from './client';

export interface LoginRequest {
	email: string;
	password: string;
}

export interface AuthResponse {
	access_token: string;
	refresh_token: string;
	token_type: string;
}

export interface UserResponse {
	id: string;
	email: string;
	role: string;
	is_active: boolean;
	created_at: string;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
	const result = await apiFetch<AuthResponse>('/auth/login', {
		method: 'POST',
		body: JSON.stringify({ email, password })
	});
	setAccessToken(result.access_token);
	localStorage.setItem('qp_refresh_token', result.refresh_token);
	return result;
}

export async function logout(): Promise<void> {
	const refreshToken = localStorage.getItem('qp_refresh_token');
	if (refreshToken) {
		try {
			await apiFetch('/auth/logout', {
				method: 'POST',
				body: JSON.stringify({ refresh_token: refreshToken })
			});
		} catch {}
		localStorage.removeItem('qp_refresh_token');
	}
	clearAccessToken();
}

export async function refreshToken(): Promise<AuthResponse | null> {
	const rt = localStorage.getItem('qp_refresh_token');
	if (!rt) return null;
	try {
		const result = await apiFetch<AuthResponse>('/auth/refresh', {
			method: 'POST',
			body: JSON.stringify({ refresh_token: rt })
		});
		setAccessToken(result.access_token);
		localStorage.setItem('qp_refresh_token', result.refresh_token);
		return result;
	} catch {
		clearAccessToken();
		localStorage.removeItem('qp_refresh_token');
		return null;
	}
}

export async function getMe(): Promise<UserResponse> {
	return apiFetch<UserResponse>('/me');
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
	await apiFetch('/auth/password', {
		method: 'PUT',
		body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
	});
}
