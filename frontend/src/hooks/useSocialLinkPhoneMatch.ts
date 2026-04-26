// useSocialLinkPhoneMatch — Checks if a phone number belongs to an existing member and fetches their profile for merge-mode prefill.
import { useCallback, useEffect, useRef, useState } from 'react';
import { getSocialLinkPhoneMatch } from '../api/auth';
import type { SocialLinkPhoneMatchProfile } from '../types/api';

export type PhoneMatchStatus = 'idle' | 'checking' | 'matched' | 'unmatched' | 'error';

interface Options {
  token: string;
  phone: string;
  debounceMs?: number;
}

interface Result {
  status: PhoneMatchStatus;
  profile: SocialLinkPhoneMatchProfile | null;
  /** Call on input's onBlur to trigger an immediate check. */
  onBlur: () => void;
  /** Force-refetch on demand (e.g. banner confirm click). */
  refetch: () => Promise<SocialLinkPhoneMatchProfile | null>;
}

const isValidPhone = (v: string) => v.replace(/\D/g, '').length >= 9;

export function useSocialLinkPhoneMatch({ token, phone, debounceMs = 500 }: Options): Result {
  const [status, setStatus] = useState<PhoneMatchStatus>('idle');
  const [profile, setProfile] = useState<SocialLinkPhoneMatchProfile | null>(null);
  const latestPhone = useRef(phone);

  // Keep ref in sync outside of render so async stale-result checks see latest phone.
  useEffect(() => {
    latestPhone.current = phone;
  }, [phone]);

  const runCheck = useCallback(
    async (p: string): Promise<SocialLinkPhoneMatchProfile | null> => {
      if (!token || !isValidPhone(p)) {
        setStatus('idle');
        setProfile(null);
        return null;
      }
      setStatus('checking');
      try {
        const res = await getSocialLinkPhoneMatch(token, p);
        if (latestPhone.current !== p) return null;
        if (res.matched && res.profile) {
          setStatus('matched');
          setProfile(res.profile);
          return res.profile;
        }
        setStatus('unmatched');
        setProfile(null);
        return null;
      } catch {
        if (latestPhone.current === p) {
          setStatus('error');
          setProfile(null);
        }
        return null;
      }
    },
    [token],
  );

  // Debounced auto-check on phone change. When phone format is invalid we skip the
  // check; the derived `displayStatus`/`displayProfile` below present 'idle' for invalid
  // formats, so no setState is needed here (avoids cascading-render effect).
  useEffect(() => {
    if (!isValidPhone(phone)) return;
    const timer = setTimeout(() => {
      void runCheck(phone);
    }, debounceMs);
    return () => clearTimeout(timer);
  }, [phone, debounceMs, runCheck]);

  const onBlur = useCallback(() => {
    void runCheck(phone);
  }, [phone, runCheck]);

  const refetch = useCallback(() => runCheck(phone), [phone, runCheck]);

  // Derived: invalid phone always presents as 'idle' / null regardless of prior result.
  const valid = isValidPhone(phone);
  const displayStatus: PhoneMatchStatus = valid ? status : 'idle';
  const displayProfile = valid ? profile : null;

  return { status: displayStatus, profile: displayProfile, onBlur, refetch };
}
