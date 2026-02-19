// useAdList — fetches the list of ads and provides mutation helpers with toast feedback
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { AdminAdListItem, AdminAdCreateRequest } from '../types/api.ts';

export function useAdList() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const query = useQuery({
    queryKey: ['admin', 'ads'],
    queryFn: () => api.get<AdminAdListItem[]>('/api/admin/ad'),
  });

  const createMutation = useMutation({
    mutationFn: (body: AdminAdCreateRequest) => api.post('/api/admin/ad', body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'ads'] });
      addToast({ variant: 'success', title: '광고가 등록되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '광고 등록 실패', description: '다시 시도해 주세요.' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (seq: number) => api.del(`/api/admin/ad/${seq}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'ads'] });
      addToast({ variant: 'success', title: '광고가 삭제되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '광고 삭제 실패', description: '다시 시도해 주세요.' });
    },
  });

  return { ...query, createAd: createMutation.mutate, deleteAd: deleteMutation.mutate };
}
