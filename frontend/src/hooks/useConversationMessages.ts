// useConversationMessages — Fetches paginated messages in a conversation thread
import { useQuery } from '@tanstack/react-query';
import { getConversationMessages } from '../api/messages';

const PAGE_SIZE = 30;

export function useConversationMessages(otherSeq: number, page: number) {
  return useQuery({
    queryKey: ['messages', 'conversation', otherSeq, page],
    queryFn: () => getConversationMessages(otherSeq, page, PAGE_SIZE),
    enabled: otherSeq > 0,
  });
}
