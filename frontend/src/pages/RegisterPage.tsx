// RegisterPage — Member registration page with admin approval notice.
import { Navigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { PageMeta } from '../components/seo/PageMeta';
import { RegisterForm } from '../components/auth/RegisterForm';
import { useBlockBack } from '../hooks/useBlockBack';
import { AuthFooterLink, AuthScreen } from '../components/auth/AuthScreen';

export function RegisterPage() {
  useBlockBack();
  const { isLoggedIn, isLoading } = useAuth();

  if (isLoading) return null;
  if (isLoggedIn) return <Navigate to="/" replace />;

  return (
    <>
      <PageMeta title="회원가입" noIndex />
      <AuthScreen
        title="회원가입 신청"
        description="가입 신청 후 관리자 승인이 완료되어야 로그인이 가능합니다."
        align="start"
      >
        <RegisterForm />

        <div className="mt-4 text-center">
          <AuthFooterLink to="/login">로그인으로 돌아가기</AuthFooterLink>
        </div>
      </AuthScreen>
    </>
  );
}
