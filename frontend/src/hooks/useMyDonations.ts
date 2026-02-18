// useMyDonations — fetches paginated user donation history with sort
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { MyDonationResponse } from '../types/donate';

export type DonationSort = 'latest' | 'amount';

const PAGE_SIZE = 20;

export function useMyDonations() {
  const [page, setPage] = useState(1);
  const [sort, setSort] = useState<DonationSort>('latest');

  const query = useQuery({
    queryKey: ['donation', 'my', page, sort],
    queryFn: () => {
      const params = new URLSearchParams({
        page: String(page),
        size: String(PAGE_SIZE),
        sort,
      });
      return api.get<MyDonationResponse>(`/api/donation/my?${params}`);
    },
  });

  const handleSortChange = (newSort: DonationSort) => {
    setSort(newSort);
    setPage(1);
  };

  return {
    ...query,
    page,
    sort,
    setPage,
    handleSortChange,
    pageSize: PAGE_SIZE,
  };
}
