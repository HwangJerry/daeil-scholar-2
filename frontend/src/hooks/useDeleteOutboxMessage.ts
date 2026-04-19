// useDeleteOutboxMessage — Mutation to delete a sent message, invalidates outbox query
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deleteMessage } from '../api/messages';

export function useDeleteOutboxMessage(amSeq: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => deleteMessage(amSeq),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages', 'outbox'] });
    },
    onError: () => {
      alert('삭제에 실패했습니다. 다시 시도해주세요.');
    },
  });
}
