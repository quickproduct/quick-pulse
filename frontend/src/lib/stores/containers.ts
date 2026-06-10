import { writable } from 'svelte/store';
import type { Container } from '../api/containers';

export const containers = writable<Container[]>([]);
export const containerFilter = writable<string>('all');
export const containerSearch = writable<string>('');
