// useCheckPhone — Debounce + onBlur phone availability check against GET /api/auth/check-phone.
import { useCallback } from 'react';
import { api } from '../api/client';
import { useFieldAvailabilityCheck } from './useFieldAvailabilityCheck';

const isFormatValid = (v: string) => v.replace(/\D/g, '').length >= 9;

export function useCheckPhone(value: string) {
  const checkFn = useCallback(async (v: string) => {
    const res = await api.get<{ available: boolean }>(
      `/api/auth/check-phone?phone=${encodeURIComponent(v)}`,
    );
    return res.available;
  }, []);

  return useFieldAvailabilityCheck({ value, isValidFormat: isFormatValid, checkFn });
}
