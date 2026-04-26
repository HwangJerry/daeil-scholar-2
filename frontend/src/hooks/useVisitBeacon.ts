// useVisitBeacon — fires a single /api/visit/beacon per session per day, and again on route change
import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { api } from '../api/client';

function sessionKey(): string {
  return `visit-beacon-${new Date().toISOString().slice(0, 10)}`;
}

export function useVisitBeacon(): void {
  const { pathname } = useLocation();

  useEffect(() => {
    const key = sessionKey();
    if (typeof window === 'undefined') return;
    if (window.sessionStorage.getItem(key)) return;
    window.sessionStorage.setItem(key, '1');
    api.post('/api/visit/beacon').catch(() => {
      // fire-and-forget
    });
  }, [pathname]);
}
