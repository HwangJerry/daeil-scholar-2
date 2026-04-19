// useDeleteInboxMessage — Mutation to delete an inbox message, invalidates inbox and badge queries
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deleteMessage } from '../api/messages';

export function useDeleteInboxMessage(amSeq: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => deleteMessage(amSeq),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages', 'inbox'] });
      queryClient.invalidateQueries({ queryKey: ['badges'] });
    },
    onError: () => {
      alert('삭제에 실패했습니다. 다시 시도해주세요.');
    },
  });
}
