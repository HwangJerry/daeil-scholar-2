// Reactive mobile/desktop breakpoint detection hook using matchMedia
import { useSyncExternalStore } from 'react';

const MOBILE_BREAKPOINT = 768;
const query = `(max-width: ${MOBILE_BREAKPOINT - 1}px)`;

function subscribe(callback: () => void) {
  const mql = window.matchMedia(query);
  mql.addEventListener('change', callback);
  return () => mql.removeEventListener('change', callback);
}

function getSnapshot() {
  return window.matchMedia(query).matches;
}

function getServerSnapshot() {
  return false;
}

export function useResponsive() {
  const isMobile = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  return { isMobile };
}
