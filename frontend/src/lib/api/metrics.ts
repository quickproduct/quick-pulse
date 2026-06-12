import { apiFetch } from './client';

export interface MetricSnapshot {
	cpu_percent: number;
	memory_percent: number;
	memory_used: number;
	memory_total: number;
	disk_percent: number;
	net_bytes_sent: number;
	net_bytes_recv: number;
	load_1m: number;
	load_5m: number;
	load_15m: number;
	process_count: number;
	uptime_seconds: number;
}

export interface MetricHistoryPoint {
	time: string;
	value: number;
}

export interface MetricHistoryResponse {
	metric: string;
	range: string;
	data: MetricHistoryPoint[];
}

export async function getMetricsSummary(): Promise<MetricSnapshot> {
	return apiFetch<MetricSnapshot>('/metrics/summary');
}

export async function getMetricsHistory(metric: string, range = '1h'): Promise<MetricHistoryResponse> {
	return apiFetch<MetricHistoryResponse>(`/metrics/history?metric=${metric}&range=${range}`);
}
