// LoginPage — Primary login page with social OAuth buttons and ID/PW text links.
import { Navigate, useSearchParams } from 'react-router-dom';
import { AuthFooterLink, AuthNotice, AuthScreen } from '../components/auth/AuthScreen';
import { KakaoLoginButton } from '../components/auth/KakaoLoginButton';
import { useAuth } from '../hooks/useAuth';

export function LoginPage() {
  const { isLoggedIn, isLoading } = useAuth();
  const [searchParams] = useSearchParams();
  const errorParam = searchParams.get('error');

  if (isLoading) return null;
  if (isLoggedIn) return <Navigate to="/" replace />;

  return (
    <AuthScreen title="대일외국어고등학교 장학회">
      {errorParam === 'pending_approval' && (
        <AuthNotice tone="warning">
          가입 신청이 접수된 계정입니다. 관리자 승인 후 로그인 가능합니다.
        </AuthNotice>
      )}

      <div className="space-y-3">
        <KakaoLoginButton />
      </div>

      <div className="mt-6 text-center">
        <AuthFooterLink to="/login/legacy">아이디로 계속하기</AuthFooterLink>
      </div>
    </AuthScreen>
  );
}
