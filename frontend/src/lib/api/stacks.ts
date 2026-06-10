import { apiFetch } from './client';

export interface Stack {
	name: string;
	project_dir: string;
	services: StackService[];
	running: number;
	total: number;
	status: string;
	services_count: number;
}

export interface StackService {
	name: string;
	container_id: string;
	status: string;
}

export interface StackActionResponse {
	success: boolean;
	message: string;
	stack_name: string;
}

export async function listStacks(): Promise<Stack[]> {
	return apiFetch<Stack[]>('/stacks');
}

export async function getStack(name: string): Promise<Stack> {
	return apiFetch<Stack>(`/stacks/${name}`);
}

export async function startStack(name: string): Promise<StackActionResponse> {
	return apiFetch<StackActionResponse>(`/stacks/${name}/start`, { method: 'POST' });
}

export async function stopStack(name: string): Promise<StackActionResponse> {
	return apiFetch<StackActionResponse>(`/stacks/${name}/stop`, { method: 'POST' });
}

export async function restartStack(name: string): Promise<StackActionResponse> {
	return apiFetch<StackActionResponse>(`/stacks/${name}/restart`, { method: 'POST' });
}

export async function getStackConfig(name: string): Promise<{ name: string; config: string }> {
	return apiFetch<{ name: string; config: string }>(`/stacks/${name}/config`);
}

export async function saveStackConfig(name: string, config: string): Promise<{ success: boolean; message: string }> {
	return apiFetch<{ success: boolean; message: string }>(`/stacks/${name}/config`, {
		method: 'POST',
		body: JSON.stringify({ config })
	});
}

export async function createStack(name: string, config: string): Promise<{ success: boolean; message: string }> {
	return apiFetch<{ success: boolean; message: string }>('/stacks', {
		method: 'POST',
		body: JSON.stringify({ name, config })
	});
}

import { getAccessToken } from './client';

export async function deployStack(
	name: string,
	onLog: (chunk: string) => void
): Promise<void> {
	const token = getAccessToken();
	const headers: Record<string, string> = {};
	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}
	const response = await fetch(`/api/v1/stacks/${name}/deploy`, {
		method: 'POST',
		headers
	});
	if (!response.ok) {
		const error = await response.json().catch(() => ({ detail: response.statusText }));
		throw new Error(error.detail || `HTTP ${response.status}`);
	}
	if (!response.body) return;
	const reader = response.body.getReader();
	const decoder = new TextDecoder();
	while (true) {
		const { value, done } = await reader.read();
		if (done) break;
		const text = decoder.decode(value, { stream: true });
		onLog(text);
	}
}
