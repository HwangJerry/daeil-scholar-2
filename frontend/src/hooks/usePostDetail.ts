// usePostDetail — React Query hook for fetching post detail data
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { NoticeDetail } from '../types/api';

export function usePostDetail(seq: string | undefined) {
  return useQuery({
    queryKey: ['feed', 'detail', seq],
    queryFn: () => api.get<NoticeDetail>(`/api/feed/${seq}`),
    enabled: !!seq,
    staleTime: 30_000,
    retry: 2,
  });
}
