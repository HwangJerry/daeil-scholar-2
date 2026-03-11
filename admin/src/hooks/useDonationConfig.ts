// useDonationConfig — fetches and updates donation config (goal, manual adjustment, note)
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
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
    onError: () => {
      addToast({ variant: 'error', title: '설정 저장 실패', description: '다시 시도해 주세요.' });
    },
  });

  return {
    ...query,
    update: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
  };
}
