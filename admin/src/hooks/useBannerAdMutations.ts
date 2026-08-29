import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { api, ApiClientError } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { AdminBannerAdSaveRequest } from '../types/api.ts';

export function useBannerAdMutations(bnSeq: number | undefined) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const onSaveSuccess = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'bannerAds'] });
    addToast({ variant: 'success', title: '배너 광고가 저장되었습니다.' });
    navigate('/banner-ad');
  };

  const onSaveError = (error: unknown) => {
    if (error instanceof ApiClientError && error.code === 'ACTIVE_CONFLICT') {
      addToast({ variant: 'error', title: '이미 활성화된 배너 광고가 있습니다.' });
      return;
    }
    addToast({ variant: 'error', title: '저장 실패', description: '네트워크 상태를 확인하고 다시 시도해 주세요.' });
  };

  const createMutation = useMutation({
    mutationFn: (body: AdminBannerAdSaveRequest) => api.post('/api/admin/banner-ad', body),
    onSuccess: onSaveSuccess,
    onError: onSaveError,
  });

  const updateMutation = useMutation({
    mutationFn: (body: AdminBannerAdSaveRequest) => api.put(`/api/admin/banner-ad/${bnSeq}`, body),
    onSuccess: onSaveSuccess,
    onError: onSaveError,
  });

  const isSaving = createMutation.isPending || updateMutation.isPending;

  const save = (formData: AdminBannerAdSaveRequest) => {
    if (bnSeq != null) {
      updateMutation.mutate(formData);
      return;
    }
    createMutation.mutate(formData);
  };

  return { save, isSaving };
}
