// useInboxMessages — Paginated inbox query
import { useQuery } from '@tanstack/react-query';
import { getInbox } from '../api/messages';

const PAGE_SIZE = 20;

export function useInboxMessages(page: number) {
  return useQuery({
    queryKey: ['messages', 'inbox', page],
    queryFn: () => getInbox(page, PAGE_SIZE),
  });
}
