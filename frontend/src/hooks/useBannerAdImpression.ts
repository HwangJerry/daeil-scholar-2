import { useEffect, useRef } from 'react';
import { useInView } from 'react-intersection-observer';
import { api } from '../api/client';

export function useBannerAdImpression(bnSeq: number) {
  const hasFired = useRef(false);
  const previousBnSeq = useRef(bnSeq);
  const { ref, inView } = useInView({ triggerOnce: true, threshold: 0.5 });

  useEffect(() => {
    if (previousBnSeq.current !== bnSeq) {
      hasFired.current = false;
      previousBnSeq.current = bnSeq;
    }

    if (inView && !hasFired.current) {
      hasFired.current = true;
      api.post(`/api/banner-ad/${bnSeq}/view`).catch(() => {});
    }
  }, [bnSeq, inView]);

  return { ref };
}
