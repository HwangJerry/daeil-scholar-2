// AdminLoginPage — ID/PW login form for admin authentication
import { useState, type KeyboardEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth.ts';
import { ApiClientError } from '../api/client.ts';
import { Input } from '../components/ui/Input.tsx';
import { Button } from '../components/ui/Button.tsx';

function errorMessage(err: unknown): string {
  if (err instanceof ApiClientError) {
    if (err.status === 429) return '잠시 후 다시 시도해주세요';
    if (err.code === 'INVALID_CREDENTIALS') return '아이디 또는 비밀번호가 올바르지 않습니다';
    if (err.code === 'PENDING_APPROVAL') return '관리자 승인 대기 중입니다';
    if (err.code === 'FORBIDDEN') return '관리자 권한이 필요합니다';
    return err.message;
  }
  return '로그인 중 오류가 발생했습니다';
}

export function AdminLoginPage() {
  const { isLoggedIn, user, isLoading, login } = useAuth();
  const navigate = useNavigate();

  const [usrId, setUsrId] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  if (!isLoading && isLoggedIn && user?.usrStatus === 'ZZZ') {
    navigate('/', { replace: true });
    return null;
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-royal-indigo border-t-transparent" />
      </div>
    );
  }

  const handleSubmit = async () => {
    if (!usrId.trim() || !password) return;
    setError('');
    setSubmitting(true);
    try {
      await login(usrId.trim(), password);
      navigate('/', { replace: true });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Enter') handleSubmit();
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm rounded-2xl bg-white p-8 shadow-lg">
        <h1 className="mb-2 text-center text-2xl font-bold text-dark-slate">관리자 로그인</h1>
        <p className="mb-6 text-center text-sm text-cool-gray">
          관리자 권한이 있는 계정으로 로그인하세요.
        </p>

        <div className="space-y-3">
          <Input
            type="text"
            placeholder="아이디"
            value={usrId}
            onChange={(e) => setUsrId(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={submitting}
            autoComplete="username"
          />
          <Input
            type="password"
            placeholder="비밀번호"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={submitting}
            autoComplete="current-password"
          />
        </div>

        {error && (
          <p className="mt-3 text-center text-sm text-red-500">{error}</p>
        )}

        <Button
          className="mt-4 w-full"
          onClick={handleSubmit}
          disabled={submitting || !usrId.trim() || !password}
        >
          {submitting ? '로그인 중...' : '로그인'}
        </Button>

        <a
          href="/"
          className="mt-4 block text-center text-sm text-cool-gray hover:text-royal-indigo"
        >
          사용자 사이트로 이동
        </a>
      </div>
    </div>
  );
}
