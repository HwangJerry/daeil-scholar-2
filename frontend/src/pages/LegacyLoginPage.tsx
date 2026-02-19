// LegacyLoginPage — ID/password login form for legacy accounts.
import { useState } from 'react';
import { Navigate, Link, useNavigate } from 'react-router-dom';
import { legacyLogin } from '../api/auth';
import { useAuth } from '../hooks/useAuth';
import { ApiClientError } from '../api/client';
import { Button } from '../components/ui/Button';

export function LegacyLoginPage() {
  const { isLoggedIn, isLoading, fetchUser } = useAuth();
  const navigate = useNavigate();

  const [usrId, setUsrId] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  if (isLoading) return null;
  if (isLoggedIn) return <Navigate to="/" replace />;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      await legacyLogin({ usrId, password });
      await fetchUser();
      navigate('/', { replace: true });
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('로그인에 실패했습니다. 다시 시도해주세요.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  const inputClass =
    'w-full rounded-lg border border-border px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20';

  return (
    <div className="flex min-h-[60vh] items-center justify-center animate-fade-in-up">
      <div className="w-full max-w-sm">
        <h1 className="mb-6 text-center text-xl font-bold text-text-primary">기존 계정 로그인</h1>

        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">아이디</label>
            <input
              type="text"
              value={usrId}
              onChange={(e) => setUsrId(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                  e.preventDefault();
                  e.currentTarget.form?.requestSubmit();
                }
              }}
              required
              className={inputClass}
              placeholder="아이디를 입력하세요"
              autoComplete="username"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">비밀번호</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                  e.preventDefault();
                  e.currentTarget.form?.requestSubmit();
                }
              }}
              required
              className={inputClass}
              placeholder="비밀번호를 입력하세요"
              autoComplete="current-password"
            />
          </div>

          {error && <p className="text-sm text-error-text">{error}</p>}

          <Button type="submit" disabled={submitting} className="w-full">
            {submitting ? '로그인 중...' : '로그인'}
          </Button>
        </form>

        <div className="mt-4 text-center">
          <Link
            to="/login"
            className="text-xs text-text-muted hover:text-text-secondary transition-colors"
          >
            카카오 로그인으로 돌아가기
          </Link>
        </div>
      </div>
    </div>
  );
}
