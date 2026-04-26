// useFieldAvailabilityCheck — Generic hook for debounce + onBlur duplicate field checking.
import { useState, useEffect, useRef, useCallback } from 'react';

export type FieldCheckStatus = 'idle' | 'checking' | 'available' | 'unavailable' | 'error';

interface Options {
  value: string;
  isValidFormat: (v: string) => boolean;
  /** Async function that returns true when the value is available (not taken). */
  checkFn: (v: string) => Promise<boolean>;
  debounceMs?: number;
}

interface Result {
  status: FieldCheckStatus;
  /** Call on input's onBlur to trigger an immediate check. */
  onBlur: () => void;
}

export function useFieldAvailabilityCheck({
  value,
  isValidFormat,
  checkFn,
  debounceMs = 500,
}: Options): Result {
  const [status, setStatus] = useState<FieldCheckStatus>('idle');
  const latestValue = useRef(value);

  // Keep ref in sync outside of render so async stale-result checks see latest value.
  useEffect(() => {
    latestValue.current = value;
  }, [value]);

  const runCheck = useCallback(
    async (v: string) => {
      if (!isValidFormat(v)) {
        setStatus('idle');
        return;
      }
      setStatus('checking');
      try {
        const available = await checkFn(v);
        // Ignore stale results if value changed while awaiting
        if (latestValue.current === v) {
          setStatus(available ? 'available' : 'unavailable');
        }
      } catch {
        if (latestValue.current === v) {
          setStatus('error');
        }
      }
    },
    [isValidFormat, checkFn],
  );

  // Debounced auto-check on value change. When format is invalid we skip the check;
  // the derived `displayStatus` below renders 'idle' for invalid formats, so no setState
  // is needed here (avoids cascading-render effect).
  useEffect(() => {
    if (!isValidFormat(value)) return;
    const timer = setTimeout(() => runCheck(value), debounceMs);
    return () => clearTimeout(timer);
  }, [value, debounceMs, isValidFormat, runCheck]);

  const onBlur = useCallback(() => {
    runCheck(value);
  }, [value, runCheck]);

  // Derived: invalid format always presents as 'idle' regardless of prior async result.
  const displayStatus: FieldCheckStatus = isValidFormat(value) ? status : 'idle';

  return { status: displayStatus, onBlur };
}
