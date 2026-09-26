// useMemberProfileUpdate — mutation to correct a member's name, contact and academic fields
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminMemberProfileInput } from '../types/api.ts';
import { useToast } from './useToast.ts';

export function useMemberProfileUpdate(seq: string | undefined) {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  return useMutation({
    mutationFn: (input: AdminMemberProfileInput) => api.put(`/api/admin/member/${seq}/profile`, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'member', seq] });
      void queryClient.invalidateQueries({ queryKey: ['admin', 'members'] });
      addToast({ variant: 'success', title: '회원 정보가 수정되었습니다.' });
    },
  });
}
