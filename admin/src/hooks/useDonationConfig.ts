// useDonationConfig — fetches current donation config and provides update mutation
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { DonationConfig, DonationConfigUpdateRequest } from '../types/api.ts';

export function useDonationConfig() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['admin', 'donation', 'config'],
    queryFn: () => api.get<DonationConfig>('/api/admin/donation/config'),
  });

  const updateMutation = useMutation({
    mutationFn: (body: DonationConfigUpdateRequest) => api.put('/api/admin/donation/config', body),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['admin', 'donation'] }),
  });

  return {
    config: query.data,
    isLoading: query.isLoading,
    updateConfig: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
  };
}
