// Composes infinite query pagination and ad deduplication to fetch feed pages
import { useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { useHeroExclusion } from './useHeroExclusion';
import { useAdDeduplication } from './useAdDeduplication';
import type { FeedItem, FeedResponse } from '../types/api';

const PAGE_SIZE = 10;

export function useFeedPagination() {
  const { heroSeq, isHeroLoaded } = useHeroExclusion();
  const { getExcludeParam, recordAds } = useAdDeduplication();

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['feed', heroSeq ?? 'no-hero'],
    queryFn: async ({ pageParam }) => {
      const params = new URLSearchParams({ size: String(PAGE_SIZE) });
      if (pageParam) params.set('cursor', pageParam);
      if (heroSeq) params.set('exclude_seq', String(heroSeq));
      const excludeAds = getExcludeParam();
      if (excludeAds) params.set('exclude_ads', excludeAds);

      const data = await api.get<FeedResponse>(`/api/feed?${params}`);
      recordAds(data.items);
      return data;
    },
    initialPageParam: '',
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.nextCursor : undefined,
    enabled: isHeroLoaded,
    staleTime: 30_000,
  });

  const items: FeedItem[] = data?.pages.flatMap((p) => p.items) ?? [];

  return {
    items,
    hasMore: hasNextPage ?? false,
    isFetching: isFetching || isFetchingNextPage,
    loadMore: fetchNextPage,
  };
}
