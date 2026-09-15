// useAppClientBuilds — observed build numbers for one platform's threshold picker
import { useQuery } from '@tanstack/react-query';
import { fetchAppClientBuilds } from '../api/appUpdate.ts';
import type { AppPlatform } from '../types/appUpdate.ts';

export function appClientBuildsQueryKey(platform: AppPlatform) {
  return ['admin', 'app-update', 'builds', platform] as const;
}

export function useAppClientBuilds(platform: AppPlatform) {
  return useQuery({
    queryKey: appClientBuildsQueryKey(platform),
    queryFn: () => fetchAppClientBuilds(platform),
  });
}
