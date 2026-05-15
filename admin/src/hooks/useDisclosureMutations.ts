// useDisclosureMutations — create, update, and delete mutations for disclosure CRUD with toast feedback
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { CreateDisclosureRequest, UpdateDisclosureRequest } from '../types/api.ts';

export function useDisclosureMutations(seq: string | undefined) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'disclosures'] });

  const onSaveSuccess = () => {
    invalidate();
    addToast({ variant: 'success', title: '공시 자료가 저장되었습니다.' });
    navigate('/disclosure');
  };

  const onSaveError = () => {
    addToast({ variant: 'error', title: '저장 실패', description: '네트워크 상태를 확인하고 다시 시도해 주세요.' });
  };

  const createMutation = useMutation({
    mutationFn: (body: CreateDisclosureRequest) => api.post<{ seq: number }>('/api/admin/disclosure', body),
    onSuccess: onSaveSuccess,
    onError: onSaveError,
  });

  const updateMutation = useMutation({
    mutationFn: (body: UpdateDisclosureRequest) => api.put(`/api/admin/disclosure/${seq}`, body),
    onSuccess: onSaveSuccess,
    onError: onSaveError,
  });

  const deleteMutation = useMutation({
    mutationFn: (seqToDelete: number) => api.del(`/api/admin/disclosure/${seqToDelete}`),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '공시 자료가 삭제되었습니다.' });
      navigate('/disclosure');
    },
    onError: () => {
      addToast({ variant: 'error', title: '삭제 실패', description: '다시 시도해 주세요.' });
    },
  });

  const isSaving = createMutation.isPending || updateMutation.isPending;

  const save = (subject: string, contentMd: string, attachedFileSeqs: number[]) => {
    const payload = { subject, contentMd, attachedFileSeqs };
    if (seq) {
      updateMutation.mutate(payload);
    } else {
      createMutation.mutate(payload);
    }
  };

  const deleteDisclosure = (seqToDelete: number) => deleteMutation.mutate(seqToDelete);

  return { save, isSaving, deleteDisclosure };
}
