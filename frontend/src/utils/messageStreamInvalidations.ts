// messageStreamInvalidations.ts — React Query invalidation strategy for message stream events
import type { QueryClient } from '@tanstack/react-query';

interface MessageNewPayload {
  fromSeq: number;
  fromName?: string;
}

interface MessageReadPayload {
  readerSeq: number;
}

export function invalidateOnMessageNew(queryClient: QueryClient, rawData: string): void {
  let fromSeq: number | null = null;
  try {
    const payload = JSON.parse(rawData) as MessageNewPayload;
    if (typeof payload.fromSeq === 'number') {
      fromSeq = payload.fromSeq;
    }
  } catch {
    fromSeq = null;
  }

  queryClient.invalidateQueries({ queryKey: ['messages', 'conversations'] });
  queryClient.invalidateQueries({ queryKey: ['messages', 'inbox'] });
  queryClient.invalidateQueries({ queryKey: ['badges'] });
  if (fromSeq !== null) {
    queryClient.invalidateQueries({ queryKey: ['messages', 'conversation', fromSeq] });
  } else {
    queryClient.invalidateQueries({ queryKey: ['messages', 'conversation'] });
  }
}

export function invalidateOnMessageSent(queryClient: QueryClient): void {
  queryClient.invalidateQueries({ queryKey: ['messages', 'conversations'] });
}

export function invalidateOnMessageRead(queryClient: QueryClient, rawData: string): void {
  let readerSeq: number | null = null;
  try {
    const payload = JSON.parse(rawData) as MessageReadPayload;
    if (typeof payload.readerSeq === 'number') {
      readerSeq = payload.readerSeq;
    }
  } catch {
    readerSeq = null;
  }

  queryClient.invalidateQueries({ queryKey: ['messages', 'conversations'] });
  queryClient.invalidateQueries({ queryKey: ['messages', 'outbox'] });
  if (readerSeq !== null) {
    queryClient.invalidateQueries({ queryKey: ['messages', 'conversation', readerSeq] });
  } else {
    queryClient.invalidateQueries({ queryKey: ['messages', 'conversation'] });
  }
}
