// Tracks shown ad sequences to prevent duplicate ad display across feed pages
import { useCallback, useRef } from 'react';
import type { FeedItem } from '../types/api';

export function useAdDeduplication() {
  const shownSeqs = useRef<Set<number>>(new Set());

  const getExcludeParam = useCallback(() => {
    return [...shownSeqs.current].join(',');
  }, []);

  const recordAds = useCallback((items: FeedItem[]) => {
    items.forEach((item) => {
      if (item.type === 'ad') shownSeqs.current.add(item.maSeq);
    });
  }, []);

  return { getExcludeParam, recordAds };
}
