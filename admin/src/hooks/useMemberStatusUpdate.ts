// useMemberStatusUpdate — mutation to change a member's status
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';

export function useMemberStatusUpdate(seq: string | undefined) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (status: string) => api.put(`/api/admin/member/${seq}`, { status }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['admin', 'member', seq] }),
  });

  return { updateStatus: mutation.mutate, isUpdating: mutation.isPending };
}
