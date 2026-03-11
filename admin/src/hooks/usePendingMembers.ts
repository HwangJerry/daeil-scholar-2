// usePendingMembers — fetches and manages pending (BBB) member approval requests
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { AdminMemberListResponse } from '../types/api.ts';

export function usePendingMembers() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const query = useQuery({
    queryKey: ['admin', 'members', 'pending'],
    queryFn: () =>
      api.get<AdminMemberListResponse>('/api/admin/member?status=BBB&size=50'),
  });

  const approveMutation = useMutation({
    mutationFn: (seq: number) => api.put(`/api/admin/member/${seq}`, { status: 'CCC' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'members', 'pending'] });
      void queryClient.invalidateQueries({ queryKey: ['admin', 'dashboard'] });
      addToast({ variant: 'success', title: '승인이 완료되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '승인 실패', description: '다시 시도해 주세요.' });
    },
  });

  const rejectMutation = useMutation({
    mutationFn: (seq: number) => api.put(`/api/admin/member/${seq}`, { status: 'BAA' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'members', 'pending'] });
      void queryClient.invalidateQueries({ queryKey: ['admin', 'dashboard'] });
      addToast({ variant: 'success', title: '거절 처리되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '거절 실패', description: '다시 시도해 주세요.' });
    },
  });

  return {
    ...query,
    approve: approveMutation.mutate,
    reject: rejectMutation.mutate,
    isApproving: approveMutation.isPending,
    isRejecting: rejectMutation.isPending,
  };
}
