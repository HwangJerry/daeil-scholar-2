// LoginPage — Primary login page with Kakao OAuth buttons and legacy login link.
import { Navigate, Link } from 'react-router-dom';
import { KakaoLoginButton } from '../components/auth/KakaoLoginButton';
import { KakaoSignupButton } from '../components/auth/KakaoSignupButton';
import { useAuth } from '../hooks/useAuth';

export function LoginPage() {
  const { isLoggedIn, isLoading } = useAuth();

  if (isLoading) return null;
  if (isLoggedIn) return <Navigate to="/" replace />;

  return (
    <div className="flex min-h-[60vh] items-center justify-center animate-fade-in-up">
      <div className="w-full max-w-sm">
        <h1 className="mb-2 text-center text-xl font-bold text-text-primary">대일외국어고등학교 장학회</h1>
        <p className="mb-6 text-center text-[13px] text-text-tertiary">
          카카오 계정으로 간편하게 로그인하세요.
        </p>
        <KakaoLoginButton />
        <div className="my-4 flex items-center gap-3">
          <hr className="flex-1 border-border-default" />
          <span className="text-xs text-text-placeholder">또는</span>
          <hr className="flex-1 border-border-default" />
        </div>
        <KakaoSignupButton />
        <div className="mt-4 text-center">
          <Link
            to="/login/legacy"
            className="text-xs text-text-muted hover:text-text-secondary transition-colors"
          >
            기존 아이디로 로그인
          </Link>
        </div>
      </div>
    </div>
  );
}
