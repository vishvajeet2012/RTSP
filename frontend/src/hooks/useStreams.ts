import { useCallback, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { api, APIError } from '../services/api';
import type { AddStreamInput, BackendState, Stream } from '../types/stream';
import { errorMessage } from '../utils/helpers';

export function useStreams(accessRevision: number) {
  const [streams, setStreams] = useState<Stream[]>([]);
  const [backend, setBackend] = useState<BackendState>('checking');
  const [loading, setLoading] = useState(true);
  const [authRequired, setAuthRequired] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const [adding, setAdding] = useState(false);
  const [refreshRevision, setRefreshRevision] = useState(0);
  const generation = useRef(0);
  const locks = useRef(new Set<string>());

  useEffect(() => {
    let disposed = false;
    let timer: ReturnType<typeof setTimeout>;
    const controller = new AbortController();
    const refresh = async () => {
      const version = generation.current;
      const [health, list] = await Promise.allSettled([
        api.health(controller.signal),
        api.list(controller.signal),
      ]);
      if (disposed) return;
      setBackend(
        health.status === 'fulfilled'
          ? 'connected'
          : health.reason instanceof APIError && health.reason.status === 503
            ? 'degraded'
            : 'offline',
      );
      if (list.status === 'fulfilled') {
        if (version === generation.current) setStreams(list.value);
        setAuthRequired(false);
        setError(null);
      } else {
        const unauthorized = list.reason instanceof APIError && list.reason.status === 401;
        setAuthRequired(unauthorized);
        setError(
          unauthorized
            ? 'Enter your backend access token to open this workspace.'
            : 'Backend unavailable. Check the server and connection settings.',
        );
      }
      setLoading(false);
      timer = setTimeout(() => {
        void refresh();
      }, 3000);
    };
    void refresh();
    return () => {
      disposed = true;
      controller.abort();
      clearTimeout(timer);
    };
  }, [accessRevision, refreshRevision]);

  const refresh = useCallback(() => setRefreshRevision((v) => v + 1), []);
  const add = useCallback(
    async (input: AddStreamInput): Promise<boolean> => {
      if (locks.current.has('add')) return false;
      locks.current.add('add');
      generation.current += 1;
      setAdding(true);
      try {
        const stream = await api.add(input);
        generation.current += 1;
        setStreams((old) => [...old.filter((s) => s.id !== stream.id), stream]);
        toast.success('Stream added');
        refresh();
        return true;
      } catch (cause) {
        toast.error(errorMessage(cause));
        return false;
      } finally {
        setAdding(false);
        locks.current.delete('add');
      }
    },
    [refresh],
  );

  const operate = useCallback(
    async (id: string, action: 'remove' | 'start' | 'stop' | 'restart'): Promise<boolean> => {
      if (locks.current.has(id)) return false;
      locks.current.add(id);
      generation.current += 1;
      setBusy(new Set(locks.current));
      try {
        if (action === 'remove') {
          await api.remove(id);
          generation.current += 1;
          setStreams((old) => old.filter((s) => s.id !== id));
        } else {
          const next = await api.action(id, action);
          generation.current += 1;
          setStreams((old) => old.map((s) => (s.id === id ? next : s)));
        }
        toast.success(
          action === 'remove'
            ? 'Stream removed'
            : action === 'restart'
              ? 'Stream restarted'
              : action === 'stop'
                ? 'Source stopped for all viewers'
                : 'Source started',
        );
        refresh();
        return true;
      } catch (cause) {
        toast.error(errorMessage(cause));
        return false;
      } finally {
        locks.current.delete(id);
        setBusy(new Set(locks.current));
      }
    },
    [refresh],
  );

  return { streams, backend, loading, error, authRequired, busy, adding, add, operate, refresh };
}
