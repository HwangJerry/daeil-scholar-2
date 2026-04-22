// RealtimeProvider — Mounts the SSE message stream subscription for the active auth session
import type { ReactNode } from 'react';
import { useMessageStream } from '../../hooks/useMessageStream';

export function RealtimeProvider({ children }: { children: ReactNode }) {
  useMessageStream();
  return <>{children}</>;
}
