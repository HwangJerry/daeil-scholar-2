// Approval decisions use the submitted verification version, never legacy status codes.
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';

export interface PendingVerification {
  userSeq: number;
  userName: string;
  status: 'pending' | 'reapproval_pending';
  graduationYear: number | null;
  cohort: string | null;
  department: string | null;
  submittedAt: string | null;
  updatedAt: string;
}

export function usePendingMembers() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);
  const queryKey = ['admin', 'members', 'pending'];
  const query = useQuery({
    queryKey,
    queryFn: async () => {
      const responses = await Promise.all(['pending', 'reapproval_pending'].map((status) =>
        api.get<{ items: PendingVerification[] | null }>(`/api/admin/alumni-verifications?status=${status}`)));
      return { items: responses.flatMap((response) => response.items ?? []) };
    },
  });
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'members'] });
    void queryClient.invalidateQueries({ queryKey: ['admin', 'dashboard'] });
  };
  const failed = () => {
    refresh();
    addToast({ variant: 'error', title: '처리하지 못했습니다.', description: '신청 내용이 변경되었을 수 있습니다. 새로 불러온 내용을 확인해주세요.' });
  };
  const approveMutation = useMutation({
    mutationFn: (member: PendingVerification) => api.post(`/api/admin/alumni-verifications/${member.userSeq}/approve`, { expectedUpdatedAt: member.updatedAt }),
    onSuccess: () => { refresh(); addToast({ variant: 'success', title: '승인이 완료되었습니다.' }); },
    onError: failed,
  });
  const rejectMutation = useMutation({
    mutationFn: ({ member, reason }: { member: PendingVerification; reason: string }) =>
      api.post(`/api/admin/alumni-verifications/${member.userSeq}/reject`, { expectedUpdatedAt: member.updatedAt, reason: reason.trim() }),
    onSuccess: () => { refresh(); addToast({ variant: 'success', title: '거절 처리되었습니다.' }); },
    onError: failed,
  });
  return { ...query, approve: approveMutation.mutate, reject: rejectMutation.mutate,
    isApproving: approveMutation.isPending, isRejecting: rejectMutation.isPending };
}
