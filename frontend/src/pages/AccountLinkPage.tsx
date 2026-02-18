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
    <div className="py-8">
      <h1 className="mb-6 text-center text-lg font-bold text-dark-slate">
        카카오 계정 연결 / 회원가입
      </h1>
      <AccountLinkForm token={token} />
    </div>
  );
}
