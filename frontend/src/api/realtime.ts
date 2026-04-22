// realtime.ts — Factory for the authenticated SSE message stream connection
export function createMessageStream(): EventSource {
  return new EventSource('/api/messages/stream', { withCredentials: true });
}
