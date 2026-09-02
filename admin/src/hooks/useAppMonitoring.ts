// useAppMonitoring — React Query hooks for Sentry summaries and mobile business events
import { useQuery } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client.ts';
import type {
  MobileEventSummaryResponse,
  MobileEventType,
  MobilePlatform,
  SentryCrashSummaryResponse,
  SentryPerformanceSummaryResponse,
} from '../types/api.ts';

const SENTRY_TOP_ISSUE_LIMIT = 5;

export interface MobileEventSummaryFilters {
  from: string;
  to: string;
  platform?: MobilePlatform;
  eventType?: MobileEventType;
}

export function isSentryNotConfigured(error: unknown): boolean {
  return error instanceof ApiClientError
    && error.status === 503
    && error.code === 'SENTRY_NOT_CONFIGURED';
}

function shouldRetrySentryQuery(failureCount: number, error: Error): boolean {
  return !isSentryNotConfigured(error) && failureCount < 1;
}

export function useCrashSummary() {
  return useQuery({
    queryKey: ['admin', 'monitoring', 'crash-summary', SENTRY_TOP_ISSUE_LIMIT],
    queryFn: () => api.get<SentryCrashSummaryResponse>(
      `/api/admin/monitoring/crash-summary?limit=${SENTRY_TOP_ISSUE_LIMIT}`,
    ),
    retry: shouldRetrySentryQuery,
  });
}

export function usePerformanceSummary() {
  return useQuery({
    queryKey: ['admin', 'monitoring', 'performance-summary'],
    queryFn: () => api.get<SentryPerformanceSummaryResponse>(
      '/api/admin/monitoring/performance-summary',
    ),
    retry: shouldRetrySentryQuery,
  });
}

export function useMobileEventSummary(filters: MobileEventSummaryFilters) {
  return useQuery({
    queryKey: [
      'admin',
      'monitoring',
      'mobile-events',
      filters.from,
      filters.to,
      filters.platform ?? '',
      filters.eventType ?? '',
    ],
    queryFn: () => {
      const params = new URLSearchParams({ from: filters.from, to: filters.to });
      if (filters.platform) params.set('platform', filters.platform);
      if (filters.eventType) params.set('event_type', filters.eventType);
      return api.get<MobileEventSummaryResponse>(
        `/api/admin/mobile-events/summary?${params.toString()}`,
      );
    },
  });
}
