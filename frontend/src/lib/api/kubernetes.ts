import { apiFetch } from './client';

export interface ClusterOverview {
	nodes: number;
	nodes_ready: number;
	pods_total: number;
	pods_running: number;
	pods_pending: number;
	pods_failed: number;
	namespaces: number;
	source: 'live' | 'disconnected';
	connected: boolean;
	reason: string;
}

export interface KubeContext {
	name: string;
	cluster: string;
	server: string;
	current: boolean;
}

export interface KubePod {
	name: string;
	namespace: string;
	status: string;
	ready: string;
	restarts: number;
	age_seconds: number;
	node: string;
	cpu: string;
	memory: string;
	image: string;
}

export interface KubeDeployment {
	name: string;
	namespace: string;
	desired: number;
	ready: number;
	available: number;
	updated: number;
	age_seconds: number;
	image: string;
	strategy: string;
}

export interface KubeService {
	name: string;
	namespace: string;
	type: string;
	cluster_ip: string;
	external_ip: string | null;
	ports: Array<{ port: number; target_port: number | string; protocol: string; node_port?: number }>;
	selector: Record<string, string>;
	age_seconds: number;
}

export interface KubeNode {
	name: string;
	role: string;
	status: string;
	version: string;
	cpu: string;
	memory: string;
	os: string;
	arch: string;
	age_seconds: number;
}

export interface KubeEvent {
	name: string;
	namespace: string;
	type: 'Normal' | 'Warning';
	reason: string;
	object: string;
	message: string;
	count: number;
	age_seconds: number;
}

// Build a query string from { context, namespace } so every endpoint can
// consistently scope to a cluster + optional namespace without each call site
// reinventing the encoding.
function qs(params: Record<string, string | undefined>): string {
	const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '');
	if (entries.length === 0) return '';
	return '?' + entries.map(([k, v]) => `${k}=${encodeURIComponent(v as string)}`).join('&');
}

export const getContexts = (): Promise<KubeContext[]> => apiFetch('/kubernetes/contexts');

export const getClusterOverview = (context?: string): Promise<ClusterOverview> =>
	apiFetch(`/kubernetes/overview${qs({ context })}`);

export const getNodes = (context?: string): Promise<KubeNode[]> =>
	apiFetch(`/kubernetes/nodes${qs({ context })}`);

export const getPods = (namespace?: string, context?: string): Promise<KubePod[]> =>
	apiFetch(`/kubernetes/pods${qs({ namespace, context })}`);

export const getDeployments = (namespace?: string, context?: string): Promise<KubeDeployment[]> =>
	apiFetch(`/kubernetes/deployments${qs({ namespace, context })}`);

export const getServices = (namespace?: string, context?: string): Promise<KubeService[]> =>
	apiFetch(`/kubernetes/services${qs({ namespace, context })}`);

export const getNamespaces = (context?: string): Promise<string[]> =>
	apiFetch(`/kubernetes/namespaces${qs({ context })}`);

export const getEvents = (namespace?: string, context?: string): Promise<KubeEvent[]> =>
	apiFetch(`/kubernetes/events${qs({ namespace, context })}`);

export const getPodLogs = (
	namespace: string,
	podName: string,
	tail = 100,
	context?: string
): Promise<{ logs: string[] }> =>
	apiFetch(`/kubernetes/pods/${namespace}/${podName}/logs${qs({ tail: String(tail), context })}`);

export const deletePod = (
	namespace: string,
	podName: string,
	context?: string
): Promise<{ success: boolean; message: string }> =>
	apiFetch<{ success: boolean; message: string }>(
		`/kubernetes/pods/${namespace}/${podName}${qs({ context })}`,
		{ method: 'DELETE' }
	);

export const scaleDeployment = (
	namespace: string,
	name: string,
	replicas: number,
	context?: string
): Promise<{ success: boolean; message: string }> =>
	apiFetch<{ success: boolean; message: string }>(
		`/kubernetes/deployments/${namespace}/${name}/scale${qs({ context })}`,
		{
			method: 'POST',
			body: JSON.stringify({ replicas })
		}
	);
