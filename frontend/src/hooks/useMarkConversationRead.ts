// useMarkConversationRead — Marks all messages in a conversation as read
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { markConversationRead } from '../api/messages';

export function useMarkConversationRead() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (otherSeq: number) => markConversationRead(otherSeq),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages', 'conversations'] });
      queryClient.invalidateQueries({ queryKey: ['badges'] });
    },
    onError: (err) => {
      console.warn('Failed to mark conversation as read:', err);
    },
  });
}
