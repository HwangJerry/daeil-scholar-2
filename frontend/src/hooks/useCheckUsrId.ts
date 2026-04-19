// useCheckUsrId — Debounce + onBlur ID availability check against GET /api/auth/check-id.
import { useCallback } from 'react';
import { api } from '../api/client';
import { useFieldAvailabilityCheck } from './useFieldAvailabilityCheck';
import type { FieldCheckStatus } from './useFieldAvailabilityCheck';

export type { FieldCheckStatus as IdCheckStatus };

const USR_ID_REGEX = /^[a-zA-Z0-9]{4,20}$/;

const isFormatValid = (v: string) => USR_ID_REGEX.test(v);

export function useCheckUsrId(value: string) {
  const checkFn = useCallback(async (v: string) => {
    const res = await api.get<{ available: boolean }>(
      `/api/auth/check-id?usrId=${encodeURIComponent(v)}`,
    );
    return res.available;
  }, []);

  return useFieldAvailabilityCheck({ value, isValidFormat: isFormatValid, checkFn });
}
