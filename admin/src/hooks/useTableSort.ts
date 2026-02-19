// useTableSort — manages column sort state and provides client-side sorting for tables
import { useState, useCallback, useMemo } from 'react';

type SortDirection = 'asc' | 'desc' | null;

export interface SortState {
  column: string | null;
  direction: SortDirection;
}

export function useTableSort(initialColumn: string | null = null) {
  const [sort, setSort] = useState<SortState>({ column: initialColumn, direction: null });

  const toggleSort = useCallback((column: string) => {
    setSort((prev) => {
      if (prev.column !== column) return { column, direction: 'asc' };
      if (prev.direction === 'asc') return { column, direction: 'desc' };
      if (prev.direction === 'desc') return { column: null, direction: null };
      return { column, direction: 'asc' };
    });
  }, []);

  const getSortedItems = useCallback(
    <T,>(items: T[], accessors: Record<string, (item: T) => string | number | null>): T[] => {
      if (!sort.column || !sort.direction) return items;
      const accessor = accessors[sort.column];
      if (!accessor) return items;

      return [...items].sort((a, b) => {
        const aVal = accessor(a);
        const bVal = accessor(b);
        if (aVal == null && bVal == null) return 0;
        if (aVal == null) return 1;
        if (bVal == null) return -1;

        const cmp = typeof aVal === 'string' && typeof bVal === 'string'
          ? aVal.localeCompare(bVal, 'ko')
          : Number(aVal) - Number(bVal);

        return sort.direction === 'asc' ? cmp : -cmp;
      });
    },
    [sort],
  );

  return useMemo(() => ({ sort, toggleSort, getSortedItems }), [sort, toggleSort, getSortedItems]);
}
