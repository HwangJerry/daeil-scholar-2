// useMemberList — fetches paginated member list with search filters and pageSize support
import { useState, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminMemberListResponse } from '../types/api.ts';

export function useMemberList() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const query = useQuery({
    queryKey: ['admin', 'members', page, pageSize, search, statusFilter],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), size: String(pageSize) });
      if (search) params.set('name', search);
      if (statusFilter) params.set('status', statusFilter);
      return api.get<AdminMemberListResponse>(`/api/admin/member?${params}`);
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
    setPage,
    handleSearchChange,
    handleStatusChange,
    handlePageSizeChange,
  };
}
