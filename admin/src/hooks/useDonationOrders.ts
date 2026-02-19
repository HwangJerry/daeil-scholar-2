// useDonationOrders — paginated donation order list with filters, update mutation, and toast
import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { AdminDonationOrderListResponse, AdminDonationOrderUpdateRequest } from '../types/api.ts';

export function useDonationOrders() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [payTypeFilter, setPayTypeFilter] = useState('');
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const query = useQuery({
    queryKey: ['admin', 'donation', 'orders', page, pageSize, search, statusFilter, payTypeFilter],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), size: String(pageSize) });
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
      addToast({ variant: 'success', title: '주문이 수정되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '주문 수정 실패', description: '다시 시도해 주세요.' });
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

  const handlePageSizeChange = useCallback((size: number) => {
    setPageSize(size);
    setPage(1);
  }, []);

  return {
    ...query,
    page,
    pageSize,
    search,
    statusFilter,
    payTypeFilter,
    setPage,
    handleSearchChange,
    handleStatusChange,
    handlePayTypeChange,
    handlePageSizeChange,
    updateMutation,
  };
}
