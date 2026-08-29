// Fetches public notice pages and excludes non-notice feed records
import { useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { useHeroExclusion } from './useHeroExclusion';
import type { FeedResponse, NoticeItem } from '../types/api';

const PAGE_SIZE = 10;

export function useFeedPagination() {
  const { heroSeq, isHeroLoaded } = useHeroExclusion();

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isError,
    isFetching,
    isFetchingNextPage,
    refetch,
  } = useInfiniteQuery({
    queryKey: ['feed', heroSeq ?? 'no-hero'],
    queryFn: async ({ pageParam }) => {
      const params = new URLSearchParams({ size: String(PAGE_SIZE) });
      if (pageParam) params.set('cursor', pageParam);
      if (heroSeq) params.set('exclude_seq', String(heroSeq));

      const data = await api.get<FeedResponse>(`/api/feed?${params}`);
      return data;
    },
    initialPageParam: '',
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.nextCursor : undefined,
    enabled: isHeroLoaded,
    staleTime: 0,   // always refetch on mount for fresh counts
  });

  const items: NoticeItem[] = data?.pages.flatMap((page) =>
    page.items.filter((item): item is NoticeItem => item.type === 'notice'),
  ) ?? [];

  return {
    items,
    hasMore: hasNextPage ?? false,
    isError,
    isFetching: isFetching || isFetchingNextPage || !isHeroLoaded,
    loadMore: fetchNextPage,
    refetch,
  };
}
