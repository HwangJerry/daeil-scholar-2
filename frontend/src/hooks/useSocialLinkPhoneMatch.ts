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

const toPhoneKey = (v: string) => v.replace(/\D/g, '');
const isValidPhoneKey = (v: string) => v.length >= 9;
const isValidPhone = (v: string) => isValidPhoneKey(toPhoneKey(v));

export function useSocialLinkPhoneMatch({ token, phone, debounceMs = 500 }: Options): Result {
  const [status, setStatus] = useState<PhoneMatchStatus>('idle');
  const [profile, setProfile] = useState<SocialLinkPhoneMatchProfile | null>(null);
  const [matchedPhoneKey, setMatchedPhoneKey] = useState<string | null>(null);
  const latestPhoneKey = useRef(toPhoneKey(phone));

  // Keep ref in sync outside of render so async stale-result checks see latest phone key.
  useEffect(() => {
    latestPhoneKey.current = toPhoneKey(phone);
  }, [phone]);

  const runCheck = useCallback(
    async (p: string): Promise<SocialLinkPhoneMatchProfile | null> => {
      const requestedPhoneKey = toPhoneKey(p);
      if (!token || !isValidPhoneKey(requestedPhoneKey)) {
        setStatus('idle');
        setProfile(null);
        setMatchedPhoneKey(null);
        return null;
      }
      setStatus('checking');
      try {
        const res = await getSocialLinkPhoneMatch(token, p);
        if (latestPhoneKey.current !== requestedPhoneKey) return null;
        if (res.matched && res.profile) {
          setStatus('matched');
          setProfile(res.profile);
          setMatchedPhoneKey(requestedPhoneKey);
          return res.profile;
        }
        setStatus('unmatched');
        setProfile(null);
        setMatchedPhoneKey(null);
        return null;
      } catch {
        if (latestPhoneKey.current === requestedPhoneKey) {
          setStatus('error');
          setProfile(null);
          setMatchedPhoneKey(null);
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
  const phoneKey = toPhoneKey(phone);
  const valid = isValidPhoneKey(phoneKey);
  const isCurrentMatch = status === 'matched' && matchedPhoneKey === phoneKey;
  const displayStatus: PhoneMatchStatus = valid
    ? isCurrentMatch || status !== 'matched'
      ? status
      : 'idle'
    : 'idle';
  const displayProfile = valid && isCurrentMatch ? profile : null;

  return { status: displayStatus, profile: displayProfile, onBlur, refetch };
}
