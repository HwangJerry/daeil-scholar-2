// LoginPage — Primary login page with social OAuth buttons and ID/PW text links.
import { Navigate, Link, useSearchParams } from 'react-router-dom';
import { KakaoLoginButton } from '../components/auth/KakaoLoginButton';
import { useAuth } from '../hooks/useAuth';

export function LoginPage() {
  const { isLoggedIn, isLoading } = useAuth();
  const [searchParams] = useSearchParams();
  const errorParam = searchParams.get('error');

  if (isLoading) return null;
  if (isLoggedIn) return <Navigate to="/" replace />;

  return (
    <div className="flex min-h-[60vh] items-center justify-center animate-fade-in-up">
      <div className="w-full max-w-sm px-4">
        <h1 className="mb-6 text-center text-xl font-bold text-text-primary">대일외국어고등학교 장학회</h1>

        {errorParam === 'pending_approval' && (
          <div className="mb-4 rounded-lg bg-warning-subtle px-4 py-3 text-sm text-warning-text">
            가입 신청이 접수된 계정입니다. 관리자 승인 후 로그인 가능합니다.
          </div>
        )}

        <div className="space-y-3">
          <KakaoLoginButton />
        </div>

        <div className="mt-6 text-center">
          <Link
            to="/login/legacy"
            className="text-sm text-text-muted hover:text-text-secondary transition-colors"
          >
            아이디로 계속하기
          </Link>
        </div>
      </div>
    </div>
  );
}
