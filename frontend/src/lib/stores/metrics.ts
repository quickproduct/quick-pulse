import { writable } from 'svelte/store';
import type { MetricSnapshot } from '../api/metrics';

export const liveMetrics = writable<MetricSnapshot | null>(null);
