// usePostSiblings — React Query hook for fetching prev/next post navigation data
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { PostSiblings } from '../types/api';

export function usePostSiblings(seq: number | undefined) {
  return useQuery({
    queryKey: ['feed', 'siblings', seq],
    queryFn: () => api.get<PostSiblings>(`/api/feed/${seq}/siblings`),
    enabled: !!seq,
    staleTime: 30_000,
    retry: 1,
  });
}
