// useSubscription — React Query hooks for subscription fetch and cancellation
import { useEffect } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';

const AUTH_ERROR_STATUS = 401;
const STALE_TIME_MS = 60_000;

export interface Subscription {
  subSeq: number;
  amount: number;
  payType: string;
  status: string;
  startDate: string;
  nextBill: string;
}

interface UseSubscriptionOptions {
  onAuthError?: () => void;
}

export function useSubscription(options?: UseSubscriptionOptions) {
  const query = useQuery({
    queryKey: ['donation', 'subscription'],
    queryFn: () => api.get<Subscription | null>('/api/donation/subscription'),
    staleTime: STALE_TIME_MS,
  });

  useEffect(() => {
    if (
      query.error instanceof ApiClientError &&
      query.error.status === AUTH_ERROR_STATUS &&
      options?.onAuthError
    ) {
      options.onAuthError();
    }
  }, [query.error, options]);

  return query;
}

export function useCancelSubscription() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (subSeq: number) => api.del('/api/donation/subscription', { subSeq }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['donation', 'subscription'] });
    },
  });
}
