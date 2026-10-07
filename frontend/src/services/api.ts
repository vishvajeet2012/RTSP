import type { AddStreamInput, Health, Stream } from '../types/stream';
import { websocketBase } from '../utils/helpers';

const apiURL = (import.meta.env.VITE_API_URL?.trim() || window.location.origin).replace(/\/+$/, '');
const wsURL = websocketBase(apiURL, import.meta.env.VITE_WS_URL);
export const workspaceToken = (import.meta.env.VITE_ACCESS_TOKEN ?? '').trim();
let accessToken = '';

export class APIError extends Error {
  constructor(
    message: string,
    public readonly status: number,
  ) {
    super(message);
    this.name = 'APIError';
  }
}
export function setAccessToken(value: string): void {
  accessToken = value.trim();
}

async function request<T>(path: string, options: RequestInit = {}, authorize = true): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body) headers.set('Content-Type', 'application/json');
  if (accessToken && authorize) headers.set('Authorization', `Bearer ${accessToken}`);
  const timeout = AbortSignal.timeout(15_000);
  const signal = options.signal ? AbortSignal.any([options.signal, timeout]) : timeout;
  const response = await fetch(`${apiURL}${path}`, { ...options, headers, signal });
  if (!response.ok) {
    let message = `Backend request failed (${response.status}).`;
    try {
      const body: unknown = await response.json();
      if (body && typeof body === 'object' && 'error' in body && typeof body.error === 'string')
        message = body.error;
    } catch {
      /* Non-JSON proxy errors use the HTTP status. */
    }
    throw new APIError(message, response.status);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export const api = {
  health: (signal?: AbortSignal) => request<Health>('/api/health', { signal }, false),
  list: (signal?: AbortSignal) => request<Stream[]>('/api/streams', { signal }),
  add: (input: AddStreamInput) =>
    request<Stream>('/api/streams', { method: 'POST', body: JSON.stringify(input) }),
  remove: (id: string) =>
    request<void>(`/api/streams/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  action: (id: string, action: 'start' | 'stop' | 'restart') =>
    request<Stream>(`/api/streams/${encodeURIComponent(id)}/${action}`, { method: 'POST' }),
  viewerURL: async (id: string, signal: AbortSignal) => {
    const { ticket } = await request<{ ticket: string }>(
      `/api/streams/${encodeURIComponent(id)}/ticket`,
      { method: 'POST', signal },
    );
    return `${wsURL}/ws/streams/${encodeURIComponent(id)}?ticket=${encodeURIComponent(ticket)}`;
  },
};
