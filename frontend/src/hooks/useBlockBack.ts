// useBlockBack — Blocks browser back navigation on sensitive pages via history pushState trick
import { useEffect } from 'react';

// BrowserRouter doesn't support useBlocker, so we guard by pushing a dummy
// history entry on mount and re-pushing whenever the user hits back.
// Normal navigation through <Link> / navigate() is unaffected.
export function useBlockBack(): void {
  useEffect(() => {
    window.history.pushState(null, '', window.location.href);

    const onPopState = () => {
      window.history.pushState(null, '', window.location.href);
    };

    window.addEventListener('popstate', onPopState);
    return () => {
      window.removeEventListener('popstate', onPopState);
    };
  }, []);
}
