import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { BannerAd } from '../types/api';

const STALE_TIME_MS = 60_000;

export function useActiveBannerAd() {
  return useQuery({
    queryKey: ['banner-ad', 'active'],
    queryFn: () => api.get<BannerAd | null>('/api/banner-ad/active'),
    staleTime: STALE_TIME_MS,
    retry: 2,
  });
}
