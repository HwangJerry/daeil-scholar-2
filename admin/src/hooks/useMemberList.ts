// useMemberList — fetches paginated member list with search, status, cohort, department and join-date filters
import { useState, useCallback, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminMemberListResponse } from '../types/api.ts';

const SEARCH_DEBOUNCE_MS = 300;
const COHORT_PATTERN = /^(?:[1-9]|[1-9][0-9])$/;

export interface MemberAdvancedFilters {
  fn: string;
  dept: string;
  regFrom: string;
  regTo: string;
}

const EMPTY_FILTERS: MemberAdvancedFilters = { fn: '', dept: '', regFrom: '', regTo: '' };

/** Builds list query params, dropping incomplete values the API would reject. */
export function buildMemberListParams(
  page: number,
  pageSize: number,
  search: string,
  statusFilter: string,
  filters: MemberAdvancedFilters,
) {
  const params = new URLSearchParams({ page: String(page), size: String(pageSize) });
  if (search) params.set('q', search);
  if (statusFilter) params.set('status', statusFilter);
  const cohort = filters.fn.trim();
  if (COHORT_PATTERN.test(cohort)) params.set('fn', cohort);
  if (filters.dept) params.set('dept', filters.dept);
  const isRangeReversed = Boolean(filters.regFrom && filters.regTo && filters.regFrom > filters.regTo);
  if (!isRangeReversed) {
    if (filters.regFrom) params.set('regFrom', filters.regFrom);
    if (filters.regTo) params.set('regTo', filters.regTo);
  }
  return params;
}

export function useMemberList() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [inputValue, setInputValue] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [filters, setFilters] = useState<MemberAdvancedFilters>(EMPTY_FILTERS);
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const params = buildMemberListParams(page, pageSize, search, statusFilter, filters);
  const query = useQuery({
    queryKey: ['admin', 'members', params.toString()],
    queryFn: () => api.get<AdminMemberListResponse>(`/api/admin/member?${params}`),
  });

  const handleSearchChange = useCallback((value: string) => {
    setInputValue(value);
    if (debounceTimer.current) clearTimeout(debounceTimer.current);
    debounceTimer.current = setTimeout(() => {
      setSearch(value);
      setPage(1);
    }, SEARCH_DEBOUNCE_MS);
  }, []);

  const handleStatusChange = (value: string) => {
    setStatusFilter(value);
    setPage(1);
  };

  const handleFilterChange = useCallback((name: keyof MemberAdvancedFilters, value: string) => {
    setFilters((current) => ({ ...current, [name]: value }));
    setPage(1);
  }, []);

  const resetFilters = useCallback(() => {
    setFilters(EMPTY_FILTERS);
    setPage(1);
  }, []);

  const handlePageSizeChange = useCallback((size: number) => {
    setPageSize(size);
    setPage(1);
  }, []);

  return {
    ...query,
    page,
    pageSize,
    search,
    inputValue,
    statusFilter,
    filters,
    setPage,
    handleSearchChange,
    handleStatusChange,
    handleFilterChange,
    resetFilters,
    handlePageSizeChange,
  };
}
