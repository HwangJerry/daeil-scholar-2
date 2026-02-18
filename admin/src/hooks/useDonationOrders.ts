// useDonationOrders — fetches paginated donation order list with filters and update mutation
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminDonationOrderListResponse, AdminDonationOrderUpdateRequest } from '../types/api.ts';

export function useDonationOrders() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [payTypeFilter, setPayTypeFilter] = useState('');
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['admin', 'donation', 'orders', page, search, statusFilter, payTypeFilter],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), size: '20' });
      if (search) params.set('name', search);
      if (statusFilter) params.set('status', statusFilter);
      if (payTypeFilter) params.set('payType', payTypeFilter);
      return api.get<AdminDonationOrderListResponse>(`/api/admin/donation/orders?${params}`);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ seq, body }: { seq: number; body: AdminDonationOrderUpdateRequest }) =>
      api.put(`/api/admin/donation/orders/${seq}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'donation', 'orders'] });
    },
  });

  const handleSearchChange = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  const handleStatusChange = (value: string) => {
    setStatusFilter(value);
    setPage(1);
  };

  const handlePayTypeChange = (value: string) => {
    setPayTypeFilter(value);
    setPage(1);
  };

  return {
    ...query,
    page,
    search,
    statusFilter,
    payTypeFilter,
    setPage,
    handleSearchChange,
    handleStatusChange,
    handlePayTypeChange,
    updateMutation,
  };
}
