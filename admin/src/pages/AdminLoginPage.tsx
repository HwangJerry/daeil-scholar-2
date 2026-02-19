// AdminLoginPage — redirects to Kakao OAuth for admin authentication
import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth.ts';

export function AdminLoginPage() {
  const { isLoggedIn, user, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && isLoggedIn && user?.usrStatus === 'ZZZ') {
      navigate('/', { replace: true });
    }
  }, [isLoading, isLoggedIn, user, navigate]);

  const handleLogin = () => {
    window.location.href = '/api/auth/kakao';
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-royal-indigo border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm rounded-2xl bg-white p-8 shadow-lg">
        <h1 className="mb-2 text-center text-2xl font-bold text-dark-slate">관리자 로그인</h1>
        <p className="mb-6 text-center text-sm text-cool-gray">
          관리자 권한이 있는 계정으로 로그인하세요.
        </p>
        <button
          onClick={handleLogin}
          className="flex w-full items-center justify-center gap-2 rounded-xl bg-kakao px-4 py-3 text-sm font-medium text-kakao-text transition-colors hover:bg-kakao-hover"
        >
          카카오 로그인
        </button>
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
