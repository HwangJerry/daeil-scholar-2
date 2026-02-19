// useMemberStatusUpdate — mutation to change a member's status with toast feedback
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';

export function useMemberStatusUpdate(seq: string | undefined) {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const mutation = useMutation({
    mutationFn: (status: string) => api.put(`/api/admin/member/${seq}`, { status }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'member', seq] });
      addToast({ variant: 'success', title: '회원 상태가 변경되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '상태 변경 실패', description: '다시 시도해 주세요.' });
    },
  });

  return { updateStatus: mutation.mutate, isUpdating: mutation.isPending };
}
