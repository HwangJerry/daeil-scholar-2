// Triggers loading more items when a sentinel element enters the viewport
import { useEffect } from 'react';
import { useInView } from 'react-intersection-observer';

interface UseInfiniteScrollOptions {
  hasMore: boolean;
  isFetching: boolean;
  onLoadMore: () => void;
}

export function useInfiniteScroll({ hasMore, isFetching, onLoadMore }: UseInfiniteScrollOptions) {
  const { ref, inView } = useInView({ threshold: 0 });

  useEffect(() => {
    if (inView && hasMore && !isFetching) {
      onLoadMore();
    }
  }, [inView, hasMore, isFetching, onLoadMore]);

  return { sentinelRef: ref };
}
