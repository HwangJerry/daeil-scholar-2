// useScrollPolicy — Manages scroll reset/restore based on navigation type and a per-route allowlist
import { useEffect, useLayoutEffect, useRef } from 'react';
import { useLocation, useNavigationType } from 'react-router-dom';

export interface ScrollPolicyOptions {
  restoreAllowlist: readonly string[];
}

// Module-scope Map keeps scroll positions across navigations.
// Cleared on full page reload, which is intentional — fresh load starts at top.
const scrollPositions = new Map<string, number>();

export function useScrollPolicy({ restoreAllowlist }: ScrollPolicyOptions): void {
  const navigationType = useNavigationType();
  const location = useLocation();
  const prevKeyRef = useRef<string | null>(null);

  // Disable browser's automatic scroll restoration so our logic has full control.
  useEffect(() => {
    const previous = history.scrollRestoration;
    try {
      history.scrollRestoration = 'manual';
    } catch {
      // Some embedded browsers may throw; safe to ignore.
    }
    return () => {
      try {
        history.scrollRestoration = previous;
      } catch {
        // Ignore.
      }
    };
  }, []);

  // Save the outgoing route's scrollY right before the next route commits.
  // The cleanup runs after the new location.key triggers the effect, so we
  // capture scrollY of the page the user is leaving.
  useEffect(() => {
    prevKeyRef.current = location.key;
    return () => {
      if (prevKeyRef.current) {
        scrollPositions.set(prevKeyRef.current, window.scrollY);
      }
    };
  }, [location.key]);

  // Apply scroll policy after the new route mounts.
  // Two rAF ticks give React Query / Suspense a chance to paint before we
  // restore — otherwise the document may be shorter than the saved scrollY.
  useLayoutEffect(() => {
    if (location.hash) return;

    const isAllowed = restoreAllowlist.includes(location.pathname);
    let raf2: number | null = null;

    const raf1 = requestAnimationFrame(() => {
      raf2 = requestAnimationFrame(() => {
        if (navigationType === 'POP' && isAllowed) {
          const saved = scrollPositions.get(location.key) ?? 0;
          window.scrollTo(0, saved);
        } else {
          window.scrollTo(0, 0);
        }
      });
    });

    return () => {
      cancelAnimationFrame(raf1);
      if (raf2 !== null) cancelAnimationFrame(raf2);
    };
  }, [location.key, location.pathname, location.hash, navigationType, restoreAllowlist]);
}

export function ScrollPolicy(props: ScrollPolicyOptions): null {
  useScrollPolicy(props);
  return null;
}
