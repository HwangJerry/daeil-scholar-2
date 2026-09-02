// useAppSettings — TanStack Query hook for the administrator app settings list
import { useQuery } from '@tanstack/react-query';
import { fetchAppSettings } from '../api/appSettings.ts';

export const APP_SETTINGS_QUERY_KEY = ['admin', 'settings'] as const;

export function useAppSettings() {
  return useQuery({
    queryKey: APP_SETTINGS_QUERY_KEY,
    queryFn: fetchAppSettings,
  });
}
