// useNoticeDetail — fetches a single notice for the edit page
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { NoticeDetail } from '../types/api.ts';

export function useNoticeDetail(seq: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'notice', seq],
    queryFn: () => api.get<NoticeDetail>(`/api/admin/feed/${seq}`),
    enabled: Boolean(seq),
  });
}
