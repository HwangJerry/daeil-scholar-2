// AccountLinkPage — Kakao account linking / new registration. Mode (new|merge) is read from the URL, and merge mode also needs a phone via router state.
import { useSearchParams, useLocation, Navigate } from 'react-router-dom';
import { AuthScreen } from '../components/auth/AuthScreen';
import { AccountLinkForm } from '../components/auth/AccountLinkForm';
import { useBlockBack } from '../hooks/useBlockBack';
import type { SocialLinkMode } from '../types/api';

interface MergeState {
  phone?: string;
  email?: string;
}

export function AccountLinkPage() {
  useBlockBack();
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const token = searchParams.get('token') ?? '';
  const modeParam = searchParams.get('mode');
  const mode: SocialLinkMode = modeParam === 'merge' ? 'merge' : 'new';

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  const state = (location.state ?? null) as MergeState | null;
  const initialPhone = state?.phone ?? '';
  const initialEmail = state?.email ?? '';

  // merge mode without a phone in router state is an invalid entry point — bounce back to new mode
  if (mode === 'merge' && !initialPhone) {
    return <Navigate to={`/login/link?token=${encodeURIComponent(token)}`} replace />;
  }

  const heading = mode === 'merge' ? '통합 회원가입' : '카카오 회원가입';
  const subheading =
    mode === 'merge'
      ? '기존 회원 정보를 확인하고 추가 정보를 입력해주세요.'
      : '카카오 계정으로 회원 정보를 입력해주세요.';

  return (
    <AuthScreen title={heading} description={subheading} align="start">
      <AccountLinkForm
        token={token}
        mode={mode}
        initialPhone={initialPhone}
        initialEmail={initialEmail}
      />
    </AuthScreen>
  );
}
