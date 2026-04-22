// useMessageStream.ts — Subscribes to the SSE message stream and invalidates affected React Query caches
import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { createMessageStream } from '../api/realtime';
import { useAuth } from './useAuth';

const INITIAL_BACKOFF_MS = 1_000;
const MAX_BACKOFF_MS = 30_000;

interface MessageNewPayload {
  fromSeq: number;
  fromName?: string;
}

export function useMessageStream() {
  const queryClient = useQueryClient();
  const isLoggedIn = useAuth((s) => s.isLoggedIn);

  useEffect(() => {
    if (!isLoggedIn) return;

    let es: EventSource | null = null;
    let retryDelay = INITIAL_BACKOFF_MS;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let cancelled = false;

    const connect = () => {
      if (cancelled) return;
      es = createMessageStream();

      es.addEventListener('ready', () => {
        retryDelay = INITIAL_BACKOFF_MS;
      });

      es.addEventListener('message.new', (event) => {
        try {
          const payload = JSON.parse((event as MessageEvent).data) as MessageNewPayload;
          queryClient.invalidateQueries({ queryKey: ['messages', 'conversations'] });
          queryClient.invalidateQueries({ queryKey: ['messages', 'conversation', payload.fromSeq] });
          queryClient.invalidateQueries({ queryKey: ['messages', 'inbox'] });
          queryClient.invalidateQueries({ queryKey: ['badges'] });
        } catch {
          queryClient.invalidateQueries({ queryKey: ['messages'] });
          queryClient.invalidateQueries({ queryKey: ['badges'] });
        }
      });

      es.addEventListener('message.sent', () => {
        queryClient.invalidateQueries({ queryKey: ['messages', 'conversations'] });
      });

      es.onerror = () => {
        es?.close();
        es = null;
        if (cancelled) return;
        retryTimer = setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, MAX_BACKOFF_MS);
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (retryTimer) clearTimeout(retryTimer);
      es?.close();
      es = null;
    };
  }, [isLoggedIn, queryClient]);
}
