// feedQueryInvalidation.test.ts — guards against detail refetches that inflate hit counts
import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';
import { invalidateFeedSummaryQueries } from '../utils/feedQueryInvalidation';

function isInvalidated(queryClient: QueryClient, queryKey: readonly unknown[]) {
  return queryClient.getQueryCache().find({ queryKey, exact: true })?.state.isInvalidated;
}

describe('invalidateFeedSummaryQueries', () => {
  it('invalidates feed summary queries without invalidating detail queries', async () => {
    const queryClient = new QueryClient();

    queryClient.setQueryData(['feed', 123], { items: [] });
    queryClient.setQueryData(['feed', 'hero'], { seq: 123 });
    queryClient.setQueryData(['feed', 'detail', '123'], { seq: 123 });
    queryClient.setQueryData(['feed', 'comments', 123], []);
    queryClient.setQueryData(['feed', 'siblings', 123], {});

    await invalidateFeedSummaryQueries(queryClient);

    expect(isInvalidated(queryClient, ['feed', 123])).toBe(true);
    expect(isInvalidated(queryClient, ['feed', 'hero'])).toBe(true);
    expect(isInvalidated(queryClient, ['feed', 'detail', '123'])).toBe(false);
    expect(isInvalidated(queryClient, ['feed', 'comments', 123])).toBe(false);
    expect(isInvalidated(queryClient, ['feed', 'siblings', 123])).toBe(false);
  });
});
