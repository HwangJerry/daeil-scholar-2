// ResetPasswordPage — Token-validated password reset form with confirmation
import { useState, useEffect, useRef } from 'react';
import { Link, useSearchParams, useNavigate } from 'react-router-dom';
import { validateResetToken, confirmPasswordReset } from '../api/passwordReset';
import { ApiClientError } from '../api/client';
import { Button } from '../components/ui/Button';
import { AlertDialog } from '../components/ui/AlertDialog';
import { checkPasswordStrength } from '../hooks/usePasswordValidation';

const INPUT_CLASS =
  'w-full rounded-lg border border-border px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20';

const MIN_PASSWORD_LENGTH = 8;
const REDIRECT_DELAY_MS = 3_000;
const DEBOUNCE_MS = 500;

type TokenStatus = 'loading' | 'valid' | 'invalid';

export function ResetPasswordPage() {
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
    <div className="flex min-h-[60vh] items-center justify-center animate-fade-in-up">
      <div className="w-full max-w-sm">
        <h1 className="mb-6 text-center text-xl font-bold text-text-primary">
          비밀번호 재설정
        </h1>

        {tokenStatus === 'loading' && (
          <p className="text-center text-sm text-text-muted">확인 중...</p>
        )}

        {tokenStatus === 'invalid' && (
          <div className="space-y-4">
            <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-error-text">
              유효하지 않거나 만료된 링크입니다.
            </div>
            <div className="text-center">
              <Link
                to="/forgot-password"
                className="text-xs text-primary hover:text-primary/80 transition-colors"
              >
                비밀번호 재설정 다시 요청하기
              </Link>
            </div>
          </div>
        )}

        {tokenStatus === 'valid' && !success && (
          <>
            {userName && (
              <p className="mb-4 text-center text-sm text-text-muted">
                <span className="font-medium text-text-primary">{userName}</span>
                님의 새 비밀번호를 입력해주세요.
              </p>
            )}

            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-text-muted">
                  새 비밀번호
                </label>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => handlePasswordChange(e.target.value)}
                  onBlur={handlePasswordBlur}
                  required
                  className={INPUT_CLASS}
                  placeholder={`${MIN_PASSWORD_LENGTH}자 이상, 영문+숫자+특수문자 포함`}
                  autoComplete="new-password"
                />
                {passwordError && (
                  <p className="mt-1 text-xs text-error-text">{passwordError}</p>
                )}
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-text-muted">
                  비밀번호 확인
                </label>
                <input
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
                  className={INPUT_CLASS}
                  placeholder="비밀번호를 다시 입력하세요"
                  autoComplete="new-password"
                />
              </div>

              {error && <p className="text-sm text-error-text">{error}</p>}

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
      </div>
    </div>
  );
}
