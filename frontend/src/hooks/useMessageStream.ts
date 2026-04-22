// useMessageStream.ts — Manages the SSE message stream connection lifecycle for the logged-in user
import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { createMessageStream } from '../api/realtime';
import { useAuth } from './useAuth';
import {
  invalidateOnMessageNew,
  invalidateOnMessageSent,
} from '../utils/messageStreamInvalidations';

const INITIAL_BACKOFF_MS = 1_000;
const MAX_BACKOFF_MS = 30_000;

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
        invalidateOnMessageNew(queryClient, (event as MessageEvent).data);
      });

      es.addEventListener('message.sent', () => {
        invalidateOnMessageSent(queryClient);
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
