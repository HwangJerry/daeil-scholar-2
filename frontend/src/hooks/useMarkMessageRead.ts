// useMarkMessageRead — Mutation to mark an inbox message as read, invalidates inbox and badge queries
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { markAsRead } from '../api/messages';

export function useMarkMessageRead(amSeq: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => markAsRead(amSeq),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages', 'inbox'] });
      queryClient.invalidateQueries({ queryKey: ['badges'] });
    },
    onError: () => {
      console.warn('읽음 처리 실패');
    },
  });
}
