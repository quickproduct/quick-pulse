import { writable } from 'svelte/store';
import type { Alert, AlertRule } from '../api/alerts';

export const alerts = writable<Alert[]>([]);
export const alertRules = writable<AlertRule[]>([]);
