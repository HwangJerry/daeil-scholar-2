// useMemberStats — fetches member statistics (total, Kakao-linked, recent logins, pending)
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminMemberStats } from '../types/api.ts';

export function useMemberStats() {
  return useQuery({
    queryKey: ['admin', 'member', 'stats'],
    queryFn: () => api.get<AdminMemberStats>('/api/admin/member/stats'),
  });
}
