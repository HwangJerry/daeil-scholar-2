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
      <div className="w-full max-w-sm">
        <h1 className="mb-2 text-center text-xl font-bold text-text-primary">대일외국어고등학교 장학회</h1>
        <p className="mb-6 text-center text-[13px] text-text-tertiary">동문 커뮤니티</p>

        {errorParam === 'pending_approval' && (
          <div className="mb-4 rounded-lg bg-warning-subtle px-4 py-3 text-sm text-warning-text">
            가입 신청이 접수된 계정입니다. 관리자 승인 후 로그인 가능합니다.
          </div>
        )}

        <div className="space-y-3">
          <KakaoLoginButton />
        </div>

        <div className="mt-6 text-center space-y-2">
          <div className="flex items-center justify-center gap-1 text-sm">
            <Link
              to="/login/legacy"
              className="text-text-muted hover:text-text-secondary transition-colors"
            >
              아이디로 로그인
            </Link>
            <span className="text-text-placeholder">·</span>
            <Link
              to="/register"
              className="text-text-muted hover:text-text-secondary transition-colors"
            >
              아이디로 가입
            </Link>
            <span className="text-text-placeholder">·</span>
            <Link
              to="/forgot-password"
              className="text-text-muted hover:text-text-secondary transition-colors"
            >
              비밀번호 찾기
            </Link>
          </div>
          <p className="text-xs text-text-placeholder">
            소셜 로그인 시 기존 회원은 자동 연동됩니다.
          </p>
        </div>
      </div>
    </div>
  );
}
