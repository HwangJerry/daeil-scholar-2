// Tracks ad impression when the ad element enters the viewport
import { useEffect, useRef } from 'react';
import { useInView } from 'react-intersection-observer';
import { api } from '../api/client';

export function useAdImpression(maSeq: number) {
  const hasFired = useRef(false);
  const { ref, inView } = useInView({ triggerOnce: true, threshold: 0.5 });

  useEffect(() => {
    if (inView && !hasFired.current) {
      hasFired.current = true;
      api.post(`/api/ad/${maSeq}/view`).catch(() => {
        // fire-and-forget
      });
    }
  }, [inView, maSeq]);

  return { ref };
}
