import { writable } from 'svelte/store';
import type { Event } from '../api/events';

export const liveEvents = writable<Event[]>([]);
