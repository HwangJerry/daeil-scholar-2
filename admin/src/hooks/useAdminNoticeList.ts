// useAdminNoticeList — paginated notice list query with search filter
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminNoticeListResponse } from '../types/api.ts';

export function useAdminNoticeList() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'notices', page, search],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), size: '20' });
      if (search) params.set('search', search);
      return api.get<AdminNoticeListResponse>(`/api/admin/feed?${params}`);
    },
  });

  const handleSearchChange = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  return { data, isLoading, page, search, setPage, handleSearchChange };
}
