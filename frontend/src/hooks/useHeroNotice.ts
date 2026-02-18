// Shared hook for hero notice query — used by both HeroSection and useFeedPagination
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { HeroNotice } from '../types/api';

export function useHeroNotice() {
  return useQuery({
    queryKey: ['feed', 'hero'],
    queryFn: () => api.get<HeroNotice>('/api/feed/hero'),
    staleTime: 60_000,
    retry: 2,
  });
}
