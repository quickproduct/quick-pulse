import { apiFetch, getAccessToken } from './client';

/**
 * Public shape mirrors the backend's `logs.PublicEntry` struct. Keep them
 * in sync — fields here are all optional except `id`, `ts`, `level`,
 * `platform`, `source_id`, and `message`.
 */
export interface LogEntry {
	id: number;
	ts: number; // unix epoch ms
	level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR' | 'CRITICAL';
	platform: 'docker' | 'k8s';
	source_id: string;
	cluster?: string; // kubeconfig context name (or "docker" for local containers)
	container?: string;
	pod?: string;
	namespace?: string;
	service?: string;
	host?: string;
	env?: string;
	trace_id?: string;
	message: string;
	meta?: string;
}

export interface LogsPage {
	logs: LogEntry[];
	next_cursor: string;
}

export interface LogsFilter {
	level?: string[];
	platform?: string[];
	cluster?: string[];
	container?: string[];
	pod?: string[];
	namespace?: string[];
	service?: string[];
	env?: string[];
	q?: string;
	from?: number; // unix ms
	to?: number;
	cursor?: string;
	limit?: number;
}

export interface LogsSources {
	containers: string[];
	pods: string[];
	namespaces: string[];
	services: string[];
	envs: string[];
	clusters: string[];
	platforms: string[];
	dropped: number;
	active_streams: number;
}

export interface LogsStatsBucket {
	ts: number;
	count: number;
	by_level: Record<string, number>;
}

export interface LogsSettings {
	retention_hours: number;
	max_size_mb: number;
	sample_info: number;
	sample_debug: number;
}

/**
 * Builds a query string from a LogsFilter. Multi-valued fields are sent as
 * comma-separated values — the backend accepts either CSV or repeated
 * params, but CSV keeps URLs short and shareable.
 */
function buildQuery(f: LogsFilter): string {
	const parts: string[] = [];
	const add = (k: string, v: string | number) => {
		parts.push(`${k}=${encodeURIComponent(String(v))}`);
	};
	const addList = (k: string, vs?: string[]) => {
		if (!vs || vs.length === 0) return;
		add(k, vs.join(','));
	};
	addList('level', f.level);
	addList('platform', f.platform);
	addList('cluster', f.cluster);
	addList('container', f.container);
	addList('pod', f.pod);
	addList('namespace', f.namespace);
	addList('service', f.service);
	addList('env', f.env);
	if (f.q) add('q', f.q);
	if (f.from) add('from', f.from);
	if (f.to) add('to', f.to);
	if (f.cursor) add('cursor', f.cursor);
	if (f.limit) add('limit', f.limit);
	return parts.length ? '?' + parts.join('&') : '';
}

export const searchLogs = (f: LogsFilter): Promise<LogsPage> =>
	apiFetch(`/logs${buildQuery(f)}`);

export const getLog = (id: number): Promise<LogEntry> =>
	apiFetch(`/logs/${id}`);

export const getSources = (): Promise<LogsSources> =>
	apiFetch(`/logs/sources`);

export const getStats = (
	f: LogsFilter,
	bucket: '10s' | '1m' | '5m' | '1h' = '1m'
): Promise<LogsStatsBucket[]> => {
	const q = buildQuery(f);
	const sep = q ? '&' : '?';
	return apiFetch(`/logs/stats${q}${sep}bucket=${bucket}`);
};

export const getSettings = (): Promise<LogsSettings> =>
	apiFetch(`/logs/settings`);

export const saveSettings = (s: LogsSettings): Promise<LogsSettings> =>
	apiFetch(`/logs/settings`, {
		method: 'PUT',
		body: JSON.stringify(s)
	});

export function exportUrl(f: LogsFilter, format: 'csv' | 'json'): string {
	const q = buildQuery(f);
	const sep = q ? '&' : '?';
	const token = getAccessToken();
	const tokenParam = token ? `&token=${encodeURIComponent(token)}` : '';
	return `/api/v1/logs/export${q}${sep}format=${format}${tokenParam}`;
}
