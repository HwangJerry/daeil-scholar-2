// useNoticeMutations — create and update mutations for notice CRUD with toast feedback
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { CreateNoticeRequest, UpdateNoticeRequest } from '../types/api.ts';

export function useNoticeMutations(seq: string | undefined) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const onSaveSuccess = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'notices'] });
    addToast({ variant: 'success', title: '공지가 저장되었습니다.' });
    navigate('/notice');
  };

  const onSaveError = () => {
    addToast({ variant: 'error', title: '저장 실패', description: '네트워크 상태를 확인하고 다시 시도해 주세요.' });
  };

  const createMutation = useMutation({
    mutationFn: (body: CreateNoticeRequest) => api.post<{ seq: number }>('/api/admin/feed', body),
    onSuccess: onSaveSuccess,
    onError: onSaveError,
  });

  const updateMutation = useMutation({
    mutationFn: (body: UpdateNoticeRequest) => api.put(`/api/admin/feed/${seq}`, body),
    onSuccess: onSaveSuccess,
    onError: onSaveError,
  });

  const isSaving = createMutation.isPending || updateMutation.isPending;

  const save = (subject: string, contentMd: string, isPinned: boolean) => {
    const payload = { subject, contentMd, isPinned: isPinned ? 'Y' : 'N' };
    if (seq) {
      updateMutation.mutate(payload);
    } else {
      createMutation.mutate(payload);
    }
  };

  return { save, isSaving };
}
