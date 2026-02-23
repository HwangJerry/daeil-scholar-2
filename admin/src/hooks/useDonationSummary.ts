// useDonationSummary — fetches the public donation summary (total, donor count, goal, achievement rate)
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { DonationSummary } from '../types/api.ts';

export function useDonationSummary() {
  return useQuery({
    queryKey: ['donation', 'summary'],
    queryFn: () => api.get<DonationSummary>('/api/donation/summary'),
  });
}
