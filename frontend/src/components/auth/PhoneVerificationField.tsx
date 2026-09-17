// PhoneVerificationField — Signup SMS verification UI: request a code, enter it, and
// show the verified state. Renders below the phone field in the signup forms.
import { useState } from 'react';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import type { usePhoneVerification } from '../../hooks/usePhoneVerification';

const CODE_LENGTH = 6;

interface PhoneVerificationFieldProps {
  phone: string;
  verification: ReturnType<typeof usePhoneVerification>;
}

const formatCountdown = (seconds: number) => {
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return `${minutes}:${String(rest).padStart(2, '0')}`;
};

export function PhoneVerificationField({ phone, verification }: PhoneVerificationFieldProps) {
  const [code, setCode] = useState('');
  const { status, error, secondsLeft, isVerified, requestCode, confirmCode } = verification;

  const canRequest = phone.replace(/\D/g, '').length >= 9;
  const isBusy = status === 'sending' || status === 'confirming';

  if (isVerified) {
    return (
      <p className="mt-1 text-xs text-success-text">휴대폰 인증이 완료되었습니다.</p>
    );
  }

  const requestLabel = (() => {
    if (status === 'sending') return '발송 중...';
    if (status === 'sent' || status === 'confirming') return '재발송';
    return '인증번호 받기';
  })();

  return (
    <div className="mt-2 space-y-2">
      <div className="flex gap-2">
        <Button
          type="button"
          onClick={() => void requestCode()}
          disabled={!canRequest || isBusy}
          className="whitespace-nowrap"
        >
          {requestLabel}
        </Button>
        {(status === 'sent' || status === 'confirming') && (
          <Input
            type="tel"
            inputMode="numeric"
            value={code}
            onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, CODE_LENGTH))}
            placeholder={`인증번호 ${CODE_LENGTH}자리`}
            autoComplete="one-time-code"
          />
        )}
      </div>

      {(status === 'sent' || status === 'confirming') && (
        <div className="flex items-center gap-2">
          <Button
            type="button"
            onClick={() => void confirmCode(code)}
            disabled={code.length !== CODE_LENGTH || isBusy}
            className="whitespace-nowrap"
          >
            {status === 'confirming' ? '확인 중...' : '인증 확인'}
          </Button>
          {secondsLeft > 0 && (
            <span className="text-xs text-text-placeholder">
              남은 시간 {formatCountdown(secondsLeft)}
            </span>
          )}
        </div>
      )}

      {!canRequest && status === 'idle' && (
        <p className="text-xs text-text-placeholder">
          전화번호를 입력하면 인증번호를 받을 수 있습니다.
        </p>
      )}
      {error && <p className="text-xs text-error-text">{error}</p>}
    </div>
  );
}
