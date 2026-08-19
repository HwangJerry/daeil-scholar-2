// useDonationOrders — canonical donation order list and mutation hooks
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type {
  DonationOrder,
  DonationOrderFilters,
  DonationOrderInput,
  DonationOrderPage,
} from '../types/api.ts';
import { useToast } from './useToast.ts';

export const donationOrdersQueryKey = ['admin', 'donation', 'orders'] as const;

function createDonationOrderSearchParams(
  filters: DonationOrderFilters,
  page: number,
  size: number,
) {
  const params = new URLSearchParams({ page: String(page), size: String(size) });

  Object.entries(filters).forEach(([name, value]) => {
    const normalizedValue = value.trim();
    if (normalizedValue) params.set(name, normalizedValue);
  });

  return params;
}

export function useDonationOrdersList(
  filters: DonationOrderFilters,
  page: number,
  size: number,
) {
  return useQuery({
    queryKey: [...donationOrdersQueryKey, filters, page, size],
    queryFn: () => {
      const params = createDonationOrderSearchParams(filters, page, size);
      return api.get<DonationOrderPage>(`/api/admin/donation/orders?${params.toString()}`);
    },
  });
}

export function useCreateDonationOrder() {
  const queryClient = useQueryClient();
  const addToast = useToast((state) => state.addToast);

  return useMutation({
    mutationFn: (input: DonationOrderInput) =>
      api.post<DonationOrder>('/api/admin/donation/orders', input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: donationOrdersQueryKey });
      addToast({ variant: 'success', title: '기부 주문이 등록되었습니다.' });
    },
  });
}

interface UpdateDonationOrderVariables {
  orderSeq: number;
  input: DonationOrderInput;
}

export function useUpdateDonationOrder() {
  const queryClient = useQueryClient();
  const addToast = useToast((state) => state.addToast);

  return useMutation({
    mutationFn: ({ orderSeq, input }: UpdateDonationOrderVariables) =>
      api.put<DonationOrder>(`/api/admin/donation/orders/${orderSeq}`, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: donationOrdersQueryKey });
      addToast({ variant: 'success', title: '기부 주문이 수정되었습니다.' });
    },
  });
}
