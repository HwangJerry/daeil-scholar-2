// useDisclosureList — Infinite query for public disclosure list pagination
import { useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { DisclosureItem, DisclosureListResponse } from '../types/api';

const PAGE_SIZE = 15;

export function useDisclosureList() {
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['disclosure', 'list'],
    queryFn: async ({ pageParam }) => {
      const params = new URLSearchParams({ size: String(PAGE_SIZE) });
      if (pageParam) params.set('cursor', pageParam);
      return api.get<DisclosureListResponse>(`/api/disclosure?${params}`);
    },
    initialPageParam: '',
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.nextCursor : undefined,
    staleTime: 30_000,
  });

  const items: DisclosureItem[] = data?.pages.flatMap((p) => p.items) ?? [];

  return {
    items,
    hasMore: hasNextPage ?? false,
    isFetching: isFetching || isFetchingNextPage,
    loadMore: fetchNextPage,
  };
}
