// useDonationOrders — data fetching and mutation logic for donation orders
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';
import { usePagination } from './usePagination.ts';
import { useDonationOrderFilters } from './useDonationOrderFilters.ts';
import type {
  AdminDonationOrderListResponse,
  AdminDonationOrderUpdateRequest,
} from '../types/api.ts';

export function useDonationOrders() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const pagination = usePagination();
  const filters = useDonationOrderFilters(pagination.resetPage);

  const query = useQuery({
    queryKey: [
      'admin', 'donation', 'orders',
      pagination.page, pagination.pageSize,
      filters.nameFilter, filters.statusFilter, filters.payTypeFilter,
    ],
    queryFn: () => {
      const params = new URLSearchParams({
        page: String(pagination.page),
        size: String(pagination.pageSize),
      });
      if (filters.nameFilter) params.set('name', filters.nameFilter);
      if (filters.statusFilter) params.set('status', filters.statusFilter);
      if (filters.payTypeFilter) params.set('payType', filters.payTypeFilter);
      return api.get<AdminDonationOrderListResponse>(`/api/admin/donation/orders?${params}`);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ seq, ...body }: { seq: number } & AdminDonationOrderUpdateRequest) =>
      api.put(`/api/admin/donation/orders/${seq}`, body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'donation', 'orders'] });
      addToast({ variant: 'success', title: '주문 상태가 변경되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '상태 변경 실패', description: '다시 시도해 주세요.' });
    },
  });

  return {
    ...query,
    pagination,
    filters,
    updateOrder: updateMutation.mutate,
    isUpdatingOrder: updateMutation.isPending,
  };
}
