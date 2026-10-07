export type StreamStatus =
  'idle' | 'connecting' | 'live' | 'paused' | 'reconnecting' | 'error' | 'stopped';
export type PlayerStatus = StreamStatus | 'offline';

export interface Stream {
  id: string;
  name: string;
  /** Sanitized by the backend. Never contains the original credentials. */
  rtspUrl: string;
  hostname: string;
  status: Exclude<StreamStatus, 'paused'>;
  createdAt: string;
  startedAt?: string;
  lastError?: string;
  reconnectAttempt: number;
  viewers: number;
}

export interface AddStreamInput {
  name: string;
  rtspUrl: string;
}
export interface Health {
  status: 'ok' | 'degraded';
  ffmpeg: boolean;
}
export type BackendState = 'checking' | 'connected' | 'offline' | 'degraded';
