// usePagination — reusable pagination state (page number + page size) with reset helpers
import { useState, useCallback } from 'react';

interface UsePaginationOptions {
  initialPage?: number;
  initialPageSize?: number;
}

export function usePagination({ initialPage = 1, initialPageSize = 20 }: UsePaginationOptions = {}) {
  const [page, setPage] = useState(initialPage);
  const [pageSize, setPageSizeRaw] = useState(initialPageSize);

  const resetPage = useCallback(() => setPage(1), []);

  const handlePageSizeChange = useCallback((size: number) => {
    setPageSizeRaw(size);
    setPage(1);
  }, []);

  return { page, pageSize, setPage, resetPage, handlePageSizeChange };
}
