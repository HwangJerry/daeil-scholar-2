// useDonateOrder — mutation hook for creating donation orders with auth error handling
import { useMutation } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import type { CreateOrderRequest, CreateOrderResponse } from '../types/donate';

const AUTH_ERROR_STATUS = 401;

interface UseDonateOrderOptions {
  onAuthError?: () => void;
}

export function useDonateOrder(options?: UseDonateOrderOptions) {
  return useMutation({
    mutationFn: (req: CreateOrderRequest) =>
      api.post<CreateOrderResponse>('/api/donation/orders', req),
    onError: (error: Error) => {
      if (
        error instanceof ApiClientError &&
        error.status === AUTH_ERROR_STATUS &&
        options?.onAuthError
      ) {
        options.onAuthError();
      }
    },
  });
}

interface CreateSubscriptionRequest {
  amount: number;
  payType: 'CARD' | 'BANK' | 'HP';
}

export function useCreateSubscription(options?: UseDonateOrderOptions) {
  return useMutation({
    mutationFn: (req: CreateSubscriptionRequest) =>
      api.post<CreateOrderResponse>('/api/donation/subscription', req),
    onError: (error: Error) => {
      if (
        error instanceof ApiClientError &&
        error.status === AUTH_ERROR_STATUS &&
        options?.onAuthError
      ) {
        options.onAuthError();
      }
    },
  });
}
