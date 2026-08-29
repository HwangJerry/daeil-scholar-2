import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { AdminBannerAdListItem } from '../types/api.ts';

export function useBannerAdList() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const query = useQuery({
    queryKey: ['admin', 'bannerAds'],
    queryFn: () => api.get<AdminBannerAdListItem[]>('/api/admin/banner-ad'),
  });

  const deleteMutation = useMutation({
    mutationFn: (seq: number) => api.del(`/api/admin/banner-ad/${seq}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'bannerAds'] });
      addToast({ variant: 'success', title: '배너 광고가 삭제되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '배너 광고 삭제 실패', description: '다시 시도해 주세요.' });
    },
  });

  return {
    ...query,
    deleteAd: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
  };
}
