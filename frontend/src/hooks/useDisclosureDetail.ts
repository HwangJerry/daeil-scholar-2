// useDisclosureDetail — React Query hook for a single disclosure's full detail
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { NoticeDetail } from '../types/api';

export function useDisclosureDetail(seq: string | undefined) {
  return useQuery({
    queryKey: ['disclosure', 'detail', seq],
    queryFn: ({ signal }) => api.get<NoticeDetail>(`/api/disclosure/${seq}`, { signal }),
    enabled: !!seq,
    staleTime: 30_000,
    retry: 2,
  });
}
