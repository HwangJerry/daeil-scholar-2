// useScrollReveal — Reveals landing content once it enters the viewport
import { useEffect, useRef, useState } from 'react';

const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';
const REVEAL_ROOT_MARGIN = '0px 0px -8% 0px';
const REVEAL_THRESHOLD = 0.12;

function shouldRevealImmediately() {
  if (typeof window === 'undefined') return true;
  if (typeof IntersectionObserver === 'undefined') return true;
  if (typeof window.matchMedia !== 'function') return false;

  return window.matchMedia(REDUCED_MOTION_QUERY).matches;
}

export function useScrollReveal() {
  const elementRef = useRef<HTMLDivElement>(null);
  const [isRevealed, setIsRevealed] = useState(shouldRevealImmediately);

  useEffect(() => {
    const element = elementRef.current;
    if (!element || isRevealed) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const hasEnteredViewport = entries.some((entry) => entry.isIntersecting);
        if (!hasEnteredViewport) return;

        setIsRevealed(true);
        observer.disconnect();
      },
      {
        rootMargin: REVEAL_ROOT_MARGIN,
        threshold: REVEAL_THRESHOLD,
      },
    );

    observer.observe(element);
    return () => observer.disconnect();
  }, [isRevealed]);

  return { elementRef, isRevealed };
}
