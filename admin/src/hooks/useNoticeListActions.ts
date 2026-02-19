// useNoticeListActions — pin toggle and soft-delete mutations with toast feedback
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';

export function useNoticeListActions() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'notices'] });

  const pinMutation = useMutation({
    mutationFn: (seq: number) => api.put(`/api/admin/feed/${seq}/pin`),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '고정 상태가 변경되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '고정 변경 실패', description: '다시 시도해 주세요.' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (seq: number) => api.del(`/api/admin/feed/${seq}`),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '공지가 삭제되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '삭제 실패', description: '다시 시도해 주세요.' });
    },
  });

  return {
    togglePin: (seq: number) => pinMutation.mutate(seq),
    deleteNotice: (seq: number) => deleteMutation.mutate(seq),
  };
}
