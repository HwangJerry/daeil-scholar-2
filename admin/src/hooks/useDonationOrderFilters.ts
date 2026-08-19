// useDonationOrderFilters — canonical donation order filters with pagination reset
import { useCallback, useState } from 'react';
import type { DonationOrderFilters } from '../types/api.ts';

const INITIAL_FILTERS: DonationOrderFilters = {
  name: '',
  phone: '',
  transactionNumber: '',
  source: '',
  status: '',
  donationType: '',
};

export function useDonationOrderFilters(onFilterChange: () => void) {
  const [filters, setFilters] = useState<DonationOrderFilters>(INITIAL_FILTERS);

  const setFilter = useCallback(<Key extends keyof DonationOrderFilters>(
    name: Key,
    value: DonationOrderFilters[Key],
  ) => {
    setFilters((current) => ({ ...current, [name]: value }));
    onFilterChange();
  }, [onFilterChange]);

  const resetFilters = useCallback(() => {
    setFilters(INITIAL_FILTERS);
    onFilterChange();
  }, [onFilterChange]);

  return { filters, setFilter, resetFilters };
}
