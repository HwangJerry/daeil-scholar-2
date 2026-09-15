// useAppUpdatePolicies — TanStack Query hook for both platforms' update policies
import { useQuery } from '@tanstack/react-query';
import { fetchAppUpdatePolicies } from '../api/appUpdate.ts';

export const APP_UPDATE_POLICIES_QUERY_KEY = ['admin', 'app-update', 'policies'] as const;

export function useAppUpdatePolicies() {
  return useQuery({
    queryKey: APP_UPDATE_POLICIES_QUERY_KEY,
    queryFn: fetchAppUpdatePolicies,
  });
}
