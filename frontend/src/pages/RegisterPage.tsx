// RegisterPage — Member registration page with admin approval notice.
import { Navigate, Link } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { PageMeta } from '../components/seo/PageMeta';
import { RegisterForm } from '../components/auth/RegisterForm';
import { useBlockBack } from '../hooks/useBlockBack';

export function RegisterPage() {
  useBlockBack();
  const { isLoggedIn, isLoading } = useAuth();

  if (isLoading) return null;
  if (isLoggedIn) return <Navigate to="/" replace />;

  return (
    <>
      <PageMeta title="회원가입" noIndex />
      <div className="flex min-h-[60vh] items-start justify-center pt-10 md:pt-16 animate-fade-in-up">
      <div className="w-full max-w-sm px-4">
        <h1 className="mb-2 text-center text-xl font-bold text-text-primary">회원가입 신청</h1>
        <p className="mb-6 text-center text-[13px] text-text-tertiary">
          가입 신청 후 관리자 승인이 완료되어야 로그인이 가능합니다.
        </p>

        <RegisterForm />

        <div className="mt-4 text-center">
          <Link
            to="/login"
            className="text-xs text-text-muted hover:text-text-secondary transition-colors"
          >
            로그인으로 돌아가기
          </Link>
        </div>
      </div>
    </div>
    </>
  );
}
