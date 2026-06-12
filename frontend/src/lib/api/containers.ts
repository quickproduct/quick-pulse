import { apiFetch } from './client';

export interface Container {
	docker_id: string;
	name: string;
	image: string;
	status: string;
	ports: any[] | null;
	state?: string;
	status_text?: string;
}

export interface ContainerActionResponse {
	success: boolean;
	message: string;
	container_id: string;
}

export async function listContainers(all = false): Promise<Container[]> {
	return apiFetch<Container[]>(`/containers?all=${all}`);
}

export async function inspectContainer(id: string): Promise<any> {
	return apiFetch(`/containers/${id}`);
}

export async function startContainer(id: string): Promise<ContainerActionResponse> {
	return apiFetch<ContainerActionResponse>(`/containers/${id}/start`, { method: 'POST' });
}

export async function stopContainer(id: string): Promise<ContainerActionResponse> {
	return apiFetch<ContainerActionResponse>(`/containers/${id}/stop`, { method: 'POST' });
}

export async function restartContainer(id: string): Promise<ContainerActionResponse> {
	return apiFetch<ContainerActionResponse>(`/containers/${id}/restart`, { method: 'POST' });
}

export async function getContainerLogs(id: string, tail = 100, since?: string): Promise<{ container_id: string; logs: string[] }> {
	const params = new URLSearchParams({ tail: String(tail) });
	if (since) params.set('since', since);
	return apiFetch(`/containers/${id}/logs?${params}`);
}
