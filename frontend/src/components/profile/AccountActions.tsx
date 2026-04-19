// AccountActions — Account settings and member-since footer
import { useState } from 'react';
import { ChevronRight, Lock, LogOut } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../hooks/useAuth';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import { api } from '../../api/client';
import { PasswordChangeModal } from './PasswordChangeModal';
import type { UserProfile } from '../../types/api';

type DialogType = 'kakao' | 'nopassword' | 'password' | null;

export function AccountActions() {
  const { logout } = useAuth();
  const [dialog, setDialog] = useState<DialogType>(null);

  // Same queryKey as ProfileHeader → cache hit, no duplicate request
  const { data: profile, isLoading } = useQuery({
    queryKey: ['profile'],
    queryFn: () => api.get<UserProfile>('/api/profile'),
  });

  function handlePasswordClick() {
    if (profile?.hasPassword) {
      setDialog('password');
    } else if (profile?.hasSocialLogin) {
      setDialog('kakao');
    } else {
      setDialog('nopassword');
    }
  }

  return (
    <>
      <div className="px-4 space-y-3">
        {/* Account section */}
        <div className="rounded-[20px] bg-surface shadow-card border border-border overflow-hidden">
          <h3 className="px-5 py-3 text-sm font-semibold text-text-primary font-serif border-b border-border">
            계정
          </h3>
          <button
            onClick={handlePasswordClick}
            disabled={isLoading}
            className="w-full px-5 py-3.5 flex items-center justify-between border-b border-border hover:bg-background transition-colors disabled:opacity-50"
          >
            <div className="flex items-center gap-2.5">
              <Lock className="h-3.5 w-3.5 text-text-secondary" />
              <span className="text-sm text-text-secondary">비밀번호 변경</span>
            </div>
            <ChevronRight className="h-3.5 w-3.5 text-text-tertiary" />
          </button>
          <Button
            variant="ghost"
            className="w-full justify-start px-5 text-error hover:text-error-text hover:bg-error-light rounded-none"
            onClick={() => logout()}
          >
            <LogOut className="mr-2.5 h-3.5 w-3.5" />
            로그아웃
          </Button>
        </div>

        {/* Member since */}
        {profile?.regDate && (
          <p className="text-center text-[11px] text-text-tertiary pt-1">
            Member since {profile.regDate}
          </p>
        )}
      </div>

      {dialog === 'kakao' && (
        <Modal onClose={() => setDialog(null)} maxWidth="max-w-sm">
          <div className="p-6 space-y-4">
            <h2 className="text-base font-semibold text-text-primary font-serif">비밀번호 변경 불가</h2>
            <p className="text-sm text-text-secondary">
              카카오 로그인 계정은 여기에서 비밀번호를 변경할 수 없습니다.
            </p>
            <Button className="w-full" onClick={() => setDialog(null)}>닫기</Button>
          </div>
        </Modal>
      )}

      {dialog === 'nopassword' && (
        <Modal onClose={() => setDialog(null)} maxWidth="max-w-sm">
          <div className="p-6 space-y-4">
            <h2 className="text-base font-semibold text-text-primary font-serif">비밀번호 미설정</h2>
            <p className="text-sm text-text-secondary">
              비밀번호가 설정되어 있지 않습니다. 이메일 비밀번호 재설정을 이용해주세요.
            </p>
            <Button className="w-full" onClick={() => setDialog(null)}>닫기</Button>
          </div>
        </Modal>
      )}

      {dialog === 'password' && (
        <PasswordChangeModal onClose={() => setDialog(null)} />
      )}
    </>
  );
}
