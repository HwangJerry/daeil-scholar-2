// useAdStats — fetches per-ad view/click statistics from the admin stats endpoint
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminAdStatsItem } from '../types/api.ts';

export function useAdStats() {
  return useQuery({
    queryKey: ['admin', 'ad-stats'],
    queryFn: () => api.get<AdminAdStatsItem[]>('/api/admin/ad/stats'),
  });
}
