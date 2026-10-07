import { useEffect, useRef, useState } from 'react';
import { api, APIError } from '../services/api';
import { reconnectDelay } from '../services/websocket';

type SocketState = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'error';
interface Options {
  id: string;
  enabled: boolean;
  revision: number;
  onOpen(): void;
  onData(data: ArrayBuffer): void;
  onDisconnect(): void;
}

export function useWebSocket(options: Options): { state: SocketState; error: string | null } {
  const { id, enabled, revision } = options;
  const callbacks = useRef(options);
  callbacks.current = options;
  const [state, setState] = useState<SocketState>('idle');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled) {
      setState('idle');
      return;
    }
    let disposed = false;
    let socket: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let watchdog: ReturnType<typeof setInterval> | undefined;
    let attempts = 0;
    const controller = new AbortController();

    const retry = (message: string) => {
      if (disposed) return;
      clearInterval(watchdog);
      callbacks.current.onDisconnect();
      attempts += 1;
      if (attempts > 8) {
        setState('error');
        setError(message);
        return;
      }
      setState('reconnecting');
      retryTimer = setTimeout(() => {
        void connect();
      }, reconnectDelay(attempts));
    };

    const connect = async () => {
      if (disposed) return;
      setState(attempts ? 'reconnecting' : 'connecting');
      setError(null);
      try {
        const url = await api.viewerURL(id, controller.signal);
        if (disposed) return;
        const current = new WebSocket(url);
        socket = current;
        current.binaryType = 'arraybuffer';
        let lastData = Date.now();
        let openedAt = Date.now();
        // Includes handshakes and silent sockets; no perpetual "connecting" card.
        watchdog = setInterval(() => {
          if (Date.now() - lastData > 25_000) current.close();
        }, 1000);
        current.onopen = () => {
          if (disposed) return;
          openedAt = Date.now();
          lastData = Date.now();
          callbacks.current.onOpen();
          setState('connected');
        };
        current.onmessage = (event: MessageEvent<unknown>) => {
          if (disposed || !(event.data instanceof ArrayBuffer)) return;
          lastData = Date.now();
          if (Date.now() - openedAt > 30_000) attempts = 0;
          callbacks.current.onData(event.data);
        };
        current.onerror = () => {
          current.close();
        };
        current.onclose = () => {
          retry('Connection lost. Check the backend and restart the stream.');
        };
      } catch (cause) {
        if (disposed) return;
        if (cause instanceof APIError && [401, 403, 404, 409].includes(cause.status)) {
          callbacks.current.onDisconnect();
          setState('error');
          setError(cause.message);
          return;
        }
        retry('Unable to reach the streaming backend.');
      }
    };

    void connect();
    return () => {
      disposed = true;
      controller.abort();
      clearTimeout(retryTimer);
      clearInterval(watchdog);
      if (socket) {
        socket.onclose = null;
        socket.onerror = null;
        socket.onmessage = null;
        socket.onopen = null;
        socket.close();
      }
    };
  }, [id, enabled, revision]);

  return { state, error };
}
