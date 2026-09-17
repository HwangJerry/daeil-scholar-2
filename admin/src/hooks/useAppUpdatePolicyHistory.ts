// useAppUpdatePolicyHistory — change log for one platform, loaded when it is opened
import { useQuery } from '@tanstack/react-query';
import { fetchAppUpdatePolicyHistory } from '../api/appUpdate.ts';
import type { AppPlatform } from '../types/appUpdate.ts';

export function appUpdatePolicyHistoryQueryKey(platform: AppPlatform) {
  return ['admin', 'app-update', 'history', platform] as const;
}

export function useAppUpdatePolicyHistory(platform: AppPlatform, enabled: boolean) {
  return useQuery({
    queryKey: appUpdatePolicyHistoryQueryKey(platform),
    queryFn: () => fetchAppUpdatePolicyHistory(platform),
    enabled,
  });
}
