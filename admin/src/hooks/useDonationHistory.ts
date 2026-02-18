// useDonationHistory — fetches donation snapshot history for the last 30 days
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { DonationSnapshot } from '../types/api.ts';

export function useDonationHistory() {
  return useQuery({
    queryKey: ['admin', 'donation', 'history'],
    queryFn: () => api.get<DonationSnapshot[]>('/api/admin/donation/history'),
  });
}
