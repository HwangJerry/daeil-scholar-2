// useConversations — Fetches the list of conversation summaries for the current user
import { useQuery } from '@tanstack/react-query';
import { getConversations } from '../api/messages';

export function useConversations() {
  return useQuery({
    queryKey: ['messages', 'conversations'],
    queryFn: getConversations,
  });
}
