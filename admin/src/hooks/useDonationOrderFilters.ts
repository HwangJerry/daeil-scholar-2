// useDonationOrderFilters — filter state for donation order search (name, payment status, pay type)
import { useState, useCallback } from 'react';

export function useDonationOrderFilters(onFilterChange: () => void) {
  const [nameFilter, setNameFilterRaw] = useState('');
  const [statusFilter, setStatusFilterRaw] = useState('');
  const [payTypeFilter, setPayTypeFilterRaw] = useState('');

  const handleNameChange = useCallback((value: string) => {
    setNameFilterRaw(value);
    onFilterChange();
  }, [onFilterChange]);

  const handleStatusChange = useCallback((value: string) => {
    setStatusFilterRaw(value);
    onFilterChange();
  }, [onFilterChange]);

  const handlePayTypeChange = useCallback((value: string) => {
    setPayTypeFilterRaw(value);
    onFilterChange();
  }, [onFilterChange]);

  return {
    nameFilter,
    statusFilter,
    payTypeFilter,
    handleNameChange,
    handleStatusChange,
    handlePayTypeChange,
  };
}
