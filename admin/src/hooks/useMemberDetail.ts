// useMemberDetail — fetches a single member's detail by seq
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminMemberDetailResponse } from '../types/api.ts';

export function useMemberDetail(seq: string | undefined) {
  const query = useQuery({
    queryKey: ['admin', 'member', seq],
    queryFn: () => api.get<AdminMemberDetailResponse>(`/api/admin/member/${seq}`),
    enabled: Boolean(seq),
  });

  return { ...query, data: query.data?.member };
}
