// Shared invalidation helpers for feed summary queries.
import type { QueryClient } from '@tanstack/react-query';

const SIDE_EFFECT_DETAIL_SEGMENTS = new Set(['detail', 'comments', 'siblings']);

export function invalidateFeedSummaryQueries(queryClient: QueryClient) {
  return queryClient.invalidateQueries({
    predicate: (query) => {
      const [root, segment] = query.queryKey;
      return root === 'feed' && !SIDE_EFFECT_DETAIL_SEGMENTS.has(String(segment));
    },
  });
}
