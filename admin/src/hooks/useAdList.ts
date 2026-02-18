// useAdList — fetches the list of ads and provides mutation helpers
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminAdListItem, AdminAdCreateRequest } from '../types/api.ts';

export function useAdList() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['admin', 'ads'],
    queryFn: () => api.get<AdminAdListItem[]>('/api/admin/ad'),
  });

  const createMutation = useMutation({
    mutationFn: (body: AdminAdCreateRequest) => api.post('/api/admin/ad', body),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['admin', 'ads'] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (seq: number) => api.del(`/api/admin/ad/${seq}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['admin', 'ads'] }),
  });

  return { ...query, createAd: createMutation.mutate, deleteAd: deleteMutation.mutate };
}
