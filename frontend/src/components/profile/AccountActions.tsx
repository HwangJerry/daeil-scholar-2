// AccountActions — Account settings and member-since footer
import { useNavigate } from 'react-router-dom';
import { ChevronRight, Lock, LogOut } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../hooks/useAuth';
import { Button } from '../ui/Button';
import { api } from '../../api/client';
import type { UserProfile } from '../../types/api';

export function AccountActions() {
  const { logout } = useAuth();
  const navigate = useNavigate();

  // Same queryKey as ProfileHeader → cache hit, no duplicate request
  const { data: profile } = useQuery({
    queryKey: ['profile'],
    queryFn: () => api.get<UserProfile>('/api/profile'),
  });

  return (
    <div className="px-4 space-y-3">
      {/* Account section */}
      <div className="rounded-[20px] bg-surface shadow-card border border-border overflow-hidden">
        <h3 className="px-5 py-3 text-sm font-semibold text-text-primary font-serif border-b border-border">
          계정
        </h3>
        <button
          onClick={() => navigate('/me/password')}
          className="w-full px-5 py-3.5 flex items-center justify-between border-b border-border hover:bg-background transition-colors"
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
  );
}
