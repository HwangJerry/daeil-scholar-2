// ForgotPasswordPage — Email form to request a password reset link
import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { requestPasswordReset } from '../api/passwordReset';
import { ApiClientError } from '../api/client';
import { Button } from '../components/ui/Button';
import { AlertDialog } from '../components/ui/AlertDialog';
import { useBlockBack } from '../hooks/useBlockBack';

const INPUT_CLASS =
  'w-full rounded-lg border border-border px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20';

export function ForgotPasswordPage() {
  useBlockBack();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      await requestPasswordReset(email);
      setSuccess(true);
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('요청 처리에 실패했습니다. 다시 시도해주세요.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-[60vh] items-center justify-center animate-fade-in-up">
      <div className="w-full max-w-sm">
        <h1 className="mb-2 text-center text-xl font-bold text-text-primary">
          비밀번호 찾기
        </h1>
        <p className="mb-6 text-center text-[13px] text-text-tertiary">
          가입 시 등록한 이메일을 입력해주세요.
        </p>

        <AlertDialog
          open={success}
          title="이메일 발송 완료"
          message="입력하신 이메일로 비밀번호 재설정 링크를 보냈습니다. 메일함을 확인해주세요."
          onConfirm={() => navigate('/login/legacy', { replace: true })}
        />

        {!success && (
          <>
            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-text-muted">
                  이메일
                </label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                      e.preventDefault();
                      e.currentTarget.form?.requestSubmit();
                    }
                  }}
                  required
                  className={INPUT_CLASS}
                  placeholder="이메일을 입력하세요"
                  autoComplete="email"
                />
              </div>

              {error && <p className="text-sm text-error-text">{error}</p>}

              <Button type="submit" disabled={submitting} className="w-full">
                {submitting ? '요청 중...' : '비밀번호 재설정 링크 받기'}
              </Button>
            </form>

            <div className="mt-4 text-center">
              <Link
                to="/login/legacy"
                className="text-xs text-text-muted hover:text-text-secondary transition-colors"
              >
                로그인으로 돌아가기
              </Link>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
