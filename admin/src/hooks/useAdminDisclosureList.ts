// useAdminDisclosureList — paginated disclosure list query with search filter and pageSize support
import { useState, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminDisclosureListResponse } from '../types/api.ts';

export function useAdminDisclosureList() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'disclosures', page, pageSize, search],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), size: String(pageSize) });
      if (search) params.set('keyword', search);
      return api.get<AdminDisclosureListResponse>(`/api/admin/disclosure?${params}`);
    },
  });

  const handleSearchChange = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  const handlePageSizeChange = useCallback((size: number) => {
    setPageSize(size);
    setPage(1);
  }, []);

  return { data, isLoading, isError, refetch, page, pageSize, search, setPage, handleSearchChange, handlePageSizeChange };
}
