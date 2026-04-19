// useAdMutations — create and update mutations for ad CRUD with toast feedback
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { api, ApiClientError } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { AdminAdCreateRequest, AdminAdUpdateRequest } from '../types/api.ts';

export function useAdMutations(maSeq: number | undefined) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const onSaveSuccess = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'ads'] });
    addToast({ variant: 'success', title: '광고가 저장되었습니다.' });
    navigate('/ad');
  };

  const onSaveError = (err: unknown) => {
    if (err instanceof ApiClientError && err.code === 'TIER_CONFLICT') {
      addToast({ variant: 'error', title: '등급 충돌', description: '이미 활성화된 동일 등급의 광고가 있습니다.' });
      return;
    }
    addToast({ variant: 'error', title: '저장 실패', description: '네트워크 상태를 확인하고 다시 시도해 주세요.' });
  };

  const createMutation = useMutation({
    mutationFn: (body: AdminAdCreateRequest) => api.post('/api/admin/ad', body),
    onSuccess: onSaveSuccess,
    onError: onSaveError,
  });

  const updateMutation = useMutation({
    mutationFn: (body: AdminAdUpdateRequest) => api.put(`/api/admin/ad/${maSeq}`, body),
    onSuccess: onSaveSuccess,
    onError: onSaveError,
  });

  const isSaving = createMutation.isPending || updateMutation.isPending;

  const save = (formData: AdminAdCreateRequest | AdminAdUpdateRequest) => {
    if (maSeq != null) {
      updateMutation.mutate(formData as AdminAdUpdateRequest);
    } else {
      createMutation.mutate(formData as AdminAdCreateRequest);
    }
  };

  return { save, isSaving };
}
