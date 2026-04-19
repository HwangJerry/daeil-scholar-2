// useCheckEmail — Debounce + onBlur email availability check against GET /api/auth/check-email.
import { useCallback } from 'react';
import { api } from '../api/client';
import { useFieldAvailabilityCheck } from './useFieldAvailabilityCheck';

const isFormatValid = (v: string) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v);

export function useCheckEmail(value: string) {
  const checkFn = useCallback(async (v: string) => {
    const res = await api.get<{ available: boolean }>(
      `/api/auth/check-email?email=${encodeURIComponent(v)}`,
    );
    return res.available;
  }, []);

  return useFieldAvailabilityCheck({ value, isValidFormat: isFormatValid, checkFn });
}
