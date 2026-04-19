// useOutboxMessages — Paginated outbox query
import { useQuery } from '@tanstack/react-query';
import { getOutbox } from '../api/messages';

const PAGE_SIZE = 20;

export function useOutboxMessages(page: number) {
  return useQuery({
    queryKey: ['messages', 'outbox', page],
    queryFn: () => getOutbox(page, PAGE_SIZE),
  });
}
