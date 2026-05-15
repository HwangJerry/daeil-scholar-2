// useDisclosureDetail — fetches a single disclosure for the edit page
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { NoticeDetail } from '../types/api.ts';

export function useDisclosureDetail(seq: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'disclosure', seq],
    queryFn: () => api.get<NoticeDetail>(`/api/admin/disclosure/${seq}`),
    enabled: Boolean(seq),
  });
}
