// useHistoryList — query and mutations for history entry admin CRUD
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { HistoryEntry, HistoryUpsertRequest } from '../types/api.ts';

const QUERY_KEY = ['admin', 'history'] as const;

export function useHistoryList() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: QUERY_KEY });

  const query = useQuery<HistoryEntry[]>({
    queryKey: QUERY_KEY,
    queryFn: () => api.get<HistoryEntry[]>('/api/admin/history'),
  });

  const createMutation = useMutation({
    mutationFn: (body: HistoryUpsertRequest) =>
      api.post<{ seq: number }>('/api/admin/history', body),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '연혁이 추가되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '추가 실패', description: '다시 시도해 주세요.' });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ seq, body }: { seq: number; body: HistoryUpsertRequest }) =>
      api.put<void>(`/api/admin/history/${seq}`, body),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '저장되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '저장 실패', description: '다시 시도해 주세요.' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (seq: number) => api.del<void>(`/api/admin/history/${seq}`),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '삭제되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '삭제 실패', description: '다시 시도해 주세요.' });
    },
  });

  return {
    ...query,
    createEntry: createMutation.mutate,
    updateEntry: updateMutation.mutate,
    deleteEntry: deleteMutation.mutate,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}
