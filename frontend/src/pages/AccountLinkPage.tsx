// AccountLinkPage — Existing-member reauthentication for canonical Kakao account linking.
import { useSearchParams, Navigate } from 'react-router-dom';
import { AuthScreen } from '../components/auth/AuthScreen';
import { AccountLinkForm } from '../components/auth/AccountLinkForm';
import { useBlockBack } from '../hooks/useBlockBack';


export function AccountLinkPage() {
  useBlockBack();
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return (
    <AuthScreen
      title="카카오 계정 연결"
      description="기존 회원 계정을 확인한 뒤 카카오 로그인을 연결합니다."
      align="start"
    >
      <AccountLinkForm token={token} />
    </AuthScreen>
  );
}
