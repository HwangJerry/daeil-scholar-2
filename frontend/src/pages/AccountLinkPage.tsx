// AccountLinkPage — Kakao account linking / new registration using server-side cached token.
import { useSearchParams, Navigate } from 'react-router-dom';
import { AccountLinkForm } from '../components/auth/AccountLinkForm';

export function AccountLinkPage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="flex min-h-[60vh] items-start justify-center pt-10 md:pt-16 animate-fade-in-up">
      <div className="w-full max-w-sm px-4">
        <h1 className="mb-2 text-center text-xl font-bold text-text-primary">
          카카오 회원가입
        </h1>
        <p className="mb-6 text-center text-[13px] text-text-tertiary">
          카카오 계정으로 회원 정보를 입력해주세요.
        </p>
        <AccountLinkForm token={token} />
      </div>
    </div>
  );
}
