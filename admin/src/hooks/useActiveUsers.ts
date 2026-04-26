// useActiveUsers — React Query hook for /api/admin/stats/active-users (DAU/MAU time series)
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { ActiveUsersResponse } from '../types/api.ts';

export function useActiveUsers() {
  return useQuery({
    queryKey: ['admin', 'active-users'],
    queryFn: () => api.get<ActiveUsersResponse>('/api/admin/stats/active-users'),
    staleTime: 5 * 60 * 1000,
  });
}
