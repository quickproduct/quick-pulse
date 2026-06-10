import { apiFetch } from './client';

export interface Event {
	id: string;
	container_docker_id: string | null;
	container_name: string | null;
	event_type: string;
	timestamp: string | null;
	metadata: any;
}

export async function getEvents(limit = 50): Promise<Event[]> {
	return apiFetch<Event[]>(`/events?limit=${limit}`);
}
