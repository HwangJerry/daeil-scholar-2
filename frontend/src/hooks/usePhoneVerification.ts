// usePhoneVerification — State machine for the signup SMS verification step:
// request a code, confirm it, and hold the token signup submits.
import { useCallback, useEffect, useRef, useState } from 'react';
import { ApiClientError } from '../api/client';
import {
  confirmPhoneVerification,
  requestPhoneVerification,
} from '../api/phoneVerification';

/** 'idle' before a code is requested, 'sent' while awaiting the code, 'verified' once confirmed. */
export type PhoneVerificationStatus = 'idle' | 'sending' | 'sent' | 'confirming' | 'verified';

interface PhoneVerificationState {
  status: PhoneVerificationStatus;
  verificationId: string;
  token: string;
  verifiedPhone: string;
  error: string;
  secondsLeft: number;
}

const initialState: PhoneVerificationState = {
  status: 'idle',
  verificationId: '',
  token: '',
  verifiedPhone: '',
  error: '',
  secondsLeft: 0,
};

const errorMessage = (err: unknown, fallback: string) => {
  if (!(err instanceof ApiClientError)) return fallback;
  if (err.code === 'PHONE_VERIFICATION_THROTTLED')
    return '인증번호 요청이 너무 많습니다. 잠시 후 다시 시도해주세요.';
  if (err.code === 'PHONE_VERIFICATION_ATTEMPTS')
    return '입력 횟수를 초과했습니다. 인증번호를 다시 요청해주세요.';
  if (err.code === 'PHONE_VERIFICATION_EXPIRED')
    return '인증 시간이 만료되었습니다. 인증번호를 다시 요청해주세요.';
  if (err.code === 'PHONE_VERIFICATION_MISMATCH') return '인증번호가 일치하지 않습니다.';
  if (err.code === 'INVALID_PHONE') return '유효한 전화번호를 입력해주세요.';
  return err.message;
};

/** Drives the SMS verification step for `phone`. Changing the number clears any
 *  token already earned, so a verified token always matches the submitted number. */
export function usePhoneVerification(phone: string) {
  const [state, setState] = useState<PhoneVerificationState>(initialState);
  const tickRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const isVerified = state.status === 'verified' && state.verifiedPhone === phone;

  // A verified token belongs to one number; editing the field invalidates it.
  useEffect(() => {
    setState((prev) => {
      if (prev.status === 'idle') return prev;
      if (prev.verifiedPhone && prev.verifiedPhone !== phone) return initialState;
      return prev;
    });
  }, [phone]);

  // The remaining seconds already live in state; this only drives the tick.
  const startCountdown = useCallback(() => {
    if (tickRef.current) clearInterval(tickRef.current);
    tickRef.current = setInterval(() => {
      setState((prev) => {
        if (prev.secondsLeft <= 1) {
          if (tickRef.current) clearInterval(tickRef.current);
          return { ...prev, secondsLeft: 0 };
        }
        return { ...prev, secondsLeft: prev.secondsLeft - 1 };
      });
    }, 1000);
  }, []);

  useEffect(() => () => {
    if (tickRef.current) clearInterval(tickRef.current);
  }, []);

  const requestCode = useCallback(async () => {
    setState((prev) => ({ ...prev, status: 'sending', error: '' }));
    try {
      const result = await requestPhoneVerification(phone);
      setState({
        ...initialState,
        status: 'sent',
        verificationId: result.verificationId,
        secondsLeft: result.expiresInSec,
      });
      startCountdown();
    } catch (err) {
      setState((prev) => ({
        ...prev,
        status: 'idle',
        error: errorMessage(err, '인증번호 발송에 실패했습니다.'),
      }));
    }
  }, [phone, startCountdown]);

  const confirmCode = useCallback(
    async (code: string) => {
      setState((prev) => ({ ...prev, status: 'confirming', error: '' }));
      try {
        const result = await confirmPhoneVerification(state.verificationId, code);
        if (tickRef.current) clearInterval(tickRef.current);
        setState((prev) => ({
          ...prev,
          status: 'verified',
          token: result.verificationToken,
          verifiedPhone: phone,
          error: '',
          secondsLeft: 0,
        }));
      } catch (err) {
        setState((prev) => ({
          ...prev,
          status: 'sent',
          error: errorMessage(err, '인증 확인에 실패했습니다.'),
        }));
      }
    },
    [phone, state.verificationId],
  );

  return {
    status: state.status,
    error: state.error,
    secondsLeft: state.secondsLeft,
    isVerified,
    token: isVerified ? state.token : '',
    requestCode,
    confirmCode,
  };
}
