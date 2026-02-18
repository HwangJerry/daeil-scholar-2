// useNoticeListActions — pin toggle and soft-delete mutations for the notice list table
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';

export function useNoticeListActions() {
  const queryClient = useQueryClient();
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'notices'] });

  const pinMutation = useMutation({
    mutationFn: (seq: number) => api.put(`/api/admin/feed/${seq}/pin`),
    onSuccess: invalidate,
  });

  const deleteMutation = useMutation({
    mutationFn: (seq: number) => api.del(`/api/admin/feed/${seq}`),
    onSuccess: invalidate,
  });

  return {
    togglePin: (seq: number) => pinMutation.mutate(seq),
    deleteNotice: (seq: number) => deleteMutation.mutate(seq),
  };
}
