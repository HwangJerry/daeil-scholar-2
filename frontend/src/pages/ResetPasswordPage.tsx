// ResetPasswordPage — Token-validated password reset form with confirmation
import { useState, useEffect, useRef } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { validateResetToken, confirmPasswordReset } from '../api/passwordReset';
import { ApiClientError } from '../api/client';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { AlertDialog } from '../components/ui/AlertDialog';
import { checkPasswordStrength } from '../hooks/usePasswordValidation';
import { useBlockBack } from '../hooks/useBlockBack';
import { AuthField, AuthFieldMessage, AuthFormError } from '../components/auth/AuthFormPrimitives';
import { AuthNotice, AuthScreen, AuthTextLink } from '../components/auth/AuthScreen';

const MIN_PASSWORD_LENGTH = 8;
const REDIRECT_DELAY_MS = 3_000;
const DEBOUNCE_MS = 500;

type TokenStatus = 'loading' | 'valid' | 'invalid';

export function ResetPasswordPage() {
  useBlockBack();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const token = searchParams.get('token') ?? '';

  const [tokenStatus, setTokenStatus] = useState<TokenStatus>('loading');
  const [userName, setUserName] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const pwDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!token) {
      setTokenStatus('invalid');
      return;
    }

    validateResetToken(token)
      .then((res) => {
        if (res.valid) {
          setTokenStatus('valid');
          setUserName(res.name ?? '');
        } else {
          setTokenStatus('invalid');
        }
      })
      .catch(() => {
        setTokenStatus('invalid');
      });
  }, [token]);

  useEffect(() => {
    if (!success) return;

    const timer = setTimeout(() => {
      navigate('/login/legacy', { replace: true });
    }, REDIRECT_DELAY_MS);

    return () => clearTimeout(timer);
  }, [success, navigate]);

  const validatePw = (value: string) => {
    setPasswordError(checkPasswordStrength(value) ?? '');
  };

  const handlePasswordChange = (value: string) => {
    setNewPassword(value);
    if (pwDebounceRef.current) clearTimeout(pwDebounceRef.current);
    pwDebounceRef.current = setTimeout(() => validatePw(value), DEBOUNCE_MS);
  };

  const handlePasswordBlur = (e: React.FocusEvent<HTMLInputElement>) => {
    if (pwDebounceRef.current) clearTimeout(pwDebounceRef.current);
    validatePw(e.target.value);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    const strengthError = checkPasswordStrength(newPassword);
    if (strengthError) {
      setError(strengthError);
      return;
    }

    if (newPassword !== confirmPassword) {
      setError('비밀번호가 일치하지 않습니다.');
      return;
    }

    setSubmitting(true);

    try {
      await confirmPasswordReset(token, newPassword);
      setSuccess(true);
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('비밀번호 재설정에 실패했습니다. 다시 시도해주세요.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthScreen title="비밀번호 재설정">
      {tokenStatus === 'loading' && (
        <p className="text-center text-body-xs text-text-muted">확인 중...</p>
      )}

      {tokenStatus === 'invalid' && (
        <div className="space-y-4">
          <AuthNotice tone="error">유효하지 않거나 만료된 링크입니다.</AuthNotice>
          <div className="text-center">
            <AuthTextLink to="/forgot-password">비밀번호 재설정 다시 요청하기</AuthTextLink>
          </div>
        </div>
      )}

      {tokenStatus === 'valid' && !success && (
        <>
          {userName && (
            <p className="mb-4 text-center text-body-xs text-text-muted">
              <span className="font-medium text-text-primary">{userName}</span>
              님의 새 비밀번호를 입력해주세요.
            </p>
          )}

          <form onSubmit={handleSubmit} className="space-y-3">
            <AuthField label="새 비밀번호">
              <Input
                type="password"
                value={newPassword}
                onChange={(e) => handlePasswordChange(e.target.value)}
                onBlur={handlePasswordBlur}
                required
                placeholder={`${MIN_PASSWORD_LENGTH}자 이상, 영문+숫자+특수문자 포함`}
                autoComplete="new-password"
              />
              {passwordError && (
                <AuthFieldMessage tone="error">{passwordError}</AuthFieldMessage>
              )}
            </AuthField>

            <AuthField label="비밀번호 확인">
              <Input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                    e.preventDefault();
                    e.currentTarget.form?.requestSubmit();
                  }
                }}
                required
                placeholder="비밀번호를 다시 입력하세요"
                autoComplete="new-password"
              />
            </AuthField>

            {error && <AuthFormError>{error}</AuthFormError>}

            <Button type="submit" disabled={submitting} className="w-full">
              {submitting ? '변경 중...' : '비밀번호 변경'}
            </Button>
          </form>
        </>
      )}

      <AlertDialog
        open={tokenStatus === 'valid' && success}
        title="비밀번호 변경 완료"
        message="비밀번호가 성공적으로 변경되었습니다."
        confirmLabel="로그인하기"
        onConfirm={() => navigate('/login/legacy', { replace: true })}
      />
    </AuthScreen>
  );
}
