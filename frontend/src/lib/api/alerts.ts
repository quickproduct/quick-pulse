import { apiFetch } from './client';

export interface AlertRule {
	id: string;
	metric_type: string;
	threshold: number;
	operator: string;
	duration_seconds: number;
	enabled: boolean;
	created_at: string;
}

export interface Alert {
	id: string;
	rule_id: string | null;
	severity: string;
	message: string;
	acknowledged: boolean;
	created_at: string;
}

export async function listAlerts(): Promise<Alert[]> {
	return apiFetch<Alert[]>('/alerts');
}

export async function acknowledgeAlert(id: string): Promise<Alert> {
	return apiFetch<Alert>(`/alerts/${id}/acknowledge`, { method: 'POST' });
}

export async function listAlertRules(): Promise<AlertRule[]> {
	return apiFetch<AlertRule[]>('/alert-rules');
}

export async function createAlertRule(data: {
	metric_type: string;
	threshold: number;
	operator: string;
	duration_seconds: number;
}): Promise<AlertRule> {
	return apiFetch<AlertRule>('/alert-rules', {
		method: 'POST',
		body: JSON.stringify(data)
	});
}

export async function updateAlertRule(id: string, data: Partial<AlertRule>): Promise<AlertRule> {
	return apiFetch<AlertRule>(`/alert-rules/${id}`, {
		method: 'PUT',
		body: JSON.stringify(data)
	});
}

export async function deleteAlertRule(id: string): Promise<void> {
	await apiFetch(`/alert-rules/${id}`, { method: 'DELETE' });
}

export async function getDashboard(): Promise<any> {
	return apiFetch('/dashboard');
}
