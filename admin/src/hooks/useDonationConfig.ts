// useDonationConfig — fetches and updates donation config
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { DonationConfig, DonationConfigUpdateRequest } from '../types/api.ts';

export function useDonationConfig() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const query = useQuery({
    queryKey: ['admin', 'donation', 'config'],
    queryFn: () => api.get<DonationConfig>('/api/admin/donation/config'),
  });

  const updateMutation = useMutation({
    mutationFn: (payload: DonationConfigUpdateRequest) =>
      api.put('/api/admin/donation/config', payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'donation', 'config'] });
      void queryClient.invalidateQueries({ queryKey: ['donation', 'summary'] });
      addToast({ variant: 'success', title: '기부 설정이 저장되었습니다.' });
    },
    onError: (error) => {
      if (error instanceof ApiClientError && error.code === 'INVALID_TIER_THRESHOLDS') {
        addToast({
          variant: 'error',
          title: '나무 성장 단계 저장 실패',
          description: '각 단계 금액은 0원 이상이며 이전 단계보다 커야 합니다.',
        });
        return;
      }

      const description = error instanceof ApiClientError && error.message
        ? `서버 응답: ${error.message}`
        : '다시 시도해 주세요.';
      addToast({ variant: 'error', title: '설정 저장 실패', description });
    },
  });

  return {
    ...query,
    update: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
  };
}
