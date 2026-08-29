import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminBannerAdRow } from '../types/api.ts';

export function useBannerAdDetail(bnSeq: number | undefined) {
  return useQuery({
    queryKey: ['admin', 'bannerAds', bnSeq],
    queryFn: () => api.get<AdminBannerAdRow>(`/api/admin/banner-ad/${bnSeq}`),
    enabled: bnSeq != null,
  });
}
