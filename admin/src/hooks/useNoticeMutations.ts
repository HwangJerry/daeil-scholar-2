// useNoticeMutations — create and update mutations for notice CRUD
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client.ts';
import type { CreateNoticeRequest, UpdateNoticeRequest } from '../types/api.ts';

export function useNoticeMutations(seq: string | undefined) {
  const navigate = useNavigate();

  const createMutation = useMutation({
    mutationFn: (body: CreateNoticeRequest) => api.post<{ seq: number }>('/api/admin/feed', body),
    onSuccess: () => navigate('/notice'),
  });

  const updateMutation = useMutation({
    mutationFn: (body: UpdateNoticeRequest) => api.put(`/api/admin/feed/${seq}`, body),
    onSuccess: () => navigate('/notice'),
  });

  const isSaving = createMutation.isPending || updateMutation.isPending;

  const save = (subject: string, contentMd: string, isPinned: boolean) => {
    const payload = { subject, contentMd, isPinned: isPinned ? 'Y' : 'N' };
    if (seq) {
      updateMutation.mutate(payload);
    } else {
      createMutation.mutate({ ...payload, boardId: 'NOTICE' });
    }
  };

  return { save, isSaving };
}
