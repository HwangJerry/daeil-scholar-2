// useAdminNoticeList — paginated notice list query with search, URL-kept category filter and pageSize support
import { useState, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminNoticeListResponse } from '../types/api.ts';

export function useAdminNoticeList() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [searchParams, setSearchParams] = useSearchParams();
  const categoryParam = Number(searchParams.get('category'));
  const category = Number.isInteger(categoryParam) && categoryParam > 0 ? categoryParam : null;

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'notices', page, pageSize, search, category],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), size: String(pageSize) });
      if (search) params.set('keyword', search);
      if (category) params.set('category', String(category));
      return api.get<AdminNoticeListResponse>(`/api/admin/feed?${params}`);
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

  const handleCategoryChange = (seq: number | null) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (seq === null) next.delete('category');
      else next.set('category', String(seq));
      return next;
    });
    setPage(1);
  };

  return { data, isLoading, isError, refetch, page, pageSize, search, category, setPage, handleSearchChange, handleCategoryChange, handlePageSizeChange };
}
