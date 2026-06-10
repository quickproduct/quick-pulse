const API_BASE = '/api/v1';
const REQUEST_TIMEOUT_MS = 30_000;

let isRefreshing = false;
let refreshQueue: ((token: string) => void)[] = [];

let accessToken: string | null = null;

export function setAccessToken(token: string) {
	accessToken = token;
	if (typeof window !== 'undefined') {
		localStorage.setItem('qp_access_token', token);
	}
}

export function getAccessToken(): string | null {
	if (accessToken) return accessToken;
	if (typeof window !== 'undefined') {
		accessToken = localStorage.getItem('qp_access_token');
	}
	return accessToken;
}

export function clearAccessToken() {
	accessToken = null;
	if (typeof window !== 'undefined') {
		localStorage.removeItem('qp_access_token');
	}
}

export async function apiFetch<T>(
	endpoint: string,
	options: RequestInit = {}
): Promise<T> {
	const url = `${API_BASE}${endpoint}`;
	const token = getAccessToken();

	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options.headers as Record<string, string> || {})
	};

	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

	try {
		const response = await fetch(url, {
			...options,
			headers,
			signal: controller.signal,
		});

		if (response.status === 401 && endpoint !== '/auth/refresh' && endpoint !== '/auth/login') {
			if (typeof window !== 'undefined') {
				const rt = localStorage.getItem('qp_refresh_token');
				if (rt) {
					if (!isRefreshing) {
						isRefreshing = true;
						try {
							const refreshResponse = await fetch(`${API_BASE}/auth/refresh`, {
								method: 'POST',
								headers: { 'Content-Type': 'application/json' },
								body: JSON.stringify({ refresh_token: rt })
							});
							if (refreshResponse.ok) {
								const data = await refreshResponse.json();
								setAccessToken(data.access_token);
								localStorage.setItem('qp_refresh_token', data.refresh_token);
								isRefreshing = false;

								// Flush queue
								const oldQueue = [...refreshQueue];
								refreshQueue = [];
								oldQueue.forEach(cb => cb(data.access_token));

								// Retry original request
								headers['Authorization'] = `Bearer ${data.access_token}`;
								const retryResponse = await fetch(url, {
									...options,
									headers,
									signal: controller.signal
								});
								if (!retryResponse.ok) {
									throw new Error(`Retry failed: HTTP ${retryResponse.status}`);
								}
								if (retryResponse.status === 204) return undefined as T;
								return retryResponse.json();
							}
						} catch (refreshErr) {
							console.error('[api] Silent token refresh failed:', refreshErr);
						} finally {
							isRefreshing = false;
						}
					} else {
						// Wait for current refresh operation to finish, then retry
						return new Promise<T>((resolve, reject) => {
							refreshQueue.push((newToken) => {
								headers['Authorization'] = `Bearer ${newToken}`;
								fetch(url, { ...options, headers, signal: controller.signal })
									.then(async (res) => {
										if (!res.ok) {
											reject(new Error(`Retry failed: HTTP ${res.status}`));
										} else if (res.status === 204) {
											resolve(undefined as T);
										} else {
											resolve(res.json());
										}
									})
									.catch(reject);
							});
						});
					}
				}
			}

			// Clear session and redirect if refresh unavailable or failed
			clearAccessToken();
			if (typeof window !== 'undefined') {
				localStorage.removeItem('qp_refresh_token');
				window.location.href = '/login';
			}
			throw new Error('Unauthorized');
		}

		if (!response.ok) {
			const error = await response.json().catch(() => ({ detail: response.statusText }));
			console.error(`[api] ${options.method || 'GET'} ${endpoint} → ${response.status}`, error);
			throw new Error(error.detail || `HTTP ${response.status}`);
		}

		if (response.status === 204) return undefined as T;
		return response.json();
	} catch (err: any) {
		if (err.name === 'AbortError') {
			throw new Error('Request timed out after 30s');
		}
		throw err;
	} finally {
		clearTimeout(timeoutId);
	}
}
