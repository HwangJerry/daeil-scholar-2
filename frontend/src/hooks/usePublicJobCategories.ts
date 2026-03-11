// usePublicJobCategories.ts — Data fetching hook for public job category reference data.
import { useQuery } from '@tanstack/react-query';
import { fetchPublicJobCategories } from '../api/public';

const ONE_HOUR_MS = 60 * 60 * 1000;

/** Fetches all active job categories from the public endpoint (no auth required). */
export function usePublicJobCategories() {
  return useQuery({
    queryKey: ['public', 'job-categories'],
    queryFn: fetchPublicJobCategories,
    staleTime: ONE_HOUR_MS,
  });
}
