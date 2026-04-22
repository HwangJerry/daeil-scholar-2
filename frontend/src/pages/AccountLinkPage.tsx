// AccountLinkPage — Kakao account linking / new registration. Mode (new|merge) is read from the URL, and merge mode also needs a phone via router state.
import { useSearchParams, useLocation, Navigate } from 'react-router-dom';
import { AccountLinkForm } from '../components/auth/AccountLinkForm';
import { useBlockBack } from '../hooks/useBlockBack';
import type { SocialLinkMode } from '../types/api';

interface MergeState {
  phone?: string;
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
    <div className="flex min-h-[60vh] items-start justify-center pt-10 md:pt-16 animate-fade-in-up">
      <div className="w-full max-w-sm px-4">
        <h1 className="mb-2 text-center text-xl font-bold text-text-primary">{heading}</h1>
        <p className="mb-6 text-center text-[13px] text-text-tertiary">{subheading}</p>
        <AccountLinkForm token={token} mode={mode} initialPhone={initialPhone} />
      </div>
    </div>
  );
}
