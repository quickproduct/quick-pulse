import { getAccessToken } from '../api/client';

export class WebSocketManager {
	private connections: Map<string, WebSocket> = new Map();
	private reconnectTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();
	private messageHandlers: Map<string, Set<(data: any) => void>> = new Map();
	private reconnectAttempts: Map<string, number> = new Map();
	private paths: Map<string, string> = new Map();

	connect(channel: string, path: string) {
		if (this.connections.has(channel)) {
			return;
		}

		this.paths.set(channel, path);
		const token = getAccessToken();
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const host = window.location.host;
		const url = `${protocol}//${host}${path}?token=${token}`;

		const ws = new WebSocket(url);

		ws.onopen = () => {
			this.reconnectAttempts.set(channel, 0);
			console.debug(`[ws] connected: ${channel}`);
		};

		ws.onmessage = (event) => {
			try {
				const data = JSON.parse(event.data);
				const handlers = this.messageHandlers.get(channel);
				if (handlers) {
					for (const handler of handlers) {
						handler(data);
					}
				}
			} catch (e) {
				console.warn(`[ws] parse error on channel "${channel}":`, e);
			}
		};

		ws.onclose = (event) => {
			this.connections.delete(channel);
			const attempts = (this.reconnectAttempts.get(channel) || 0) + 1;
			this.reconnectAttempts.set(channel, attempts);
			const delay = Math.min(1000 * Math.pow(2, Math.min(attempts, 5)), 30000);
			console.debug(`[ws] closed: ${channel} (code=${event.code}), reconnecting in ${delay}ms (attempt ${attempts})`);
			const timer = setTimeout(() => {
				const savedPath = this.paths.get(channel);
				if (savedPath) this.connect(channel, savedPath);
			}, delay);
			this.reconnectTimers.set(channel, timer);
		};

		ws.onerror = (event) => {
			console.error(`[ws] error on channel "${channel}":`, event);
			ws.close();
		};

		this.connections.set(channel, ws);
	}

	disconnect(channel: string) {
		const ws = this.connections.get(channel);
		if (ws) {
			ws.close();
			this.connections.delete(channel);
		}
		const timer = this.reconnectTimers.get(channel);
		if (timer) {
			clearTimeout(timer);
			this.reconnectTimers.delete(channel);
		}
		this.messageHandlers.delete(channel);
		this.paths.delete(channel);
		this.reconnectAttempts.delete(channel);
	}

	onMessage(channel: string, handler: (data: any) => void) {
		if (!this.messageHandlers.has(channel)) {
			this.messageHandlers.set(channel, new Set());
		}
		this.messageHandlers.get(channel)!.add(handler);
		return () => {
			this.messageHandlers.get(channel)?.delete(handler);
		};
	}

	send(channel: string, data: any) {
		const ws = this.connections.get(channel);
		if (ws && ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify(data));
		}
	}

	disconnectAll() {
		for (const channel of [...this.connections.keys()]) {
			this.disconnect(channel);
		}
	}
}

export const wsManager = new WebSocketManager();
