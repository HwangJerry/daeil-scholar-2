// ProfileHeader — User profile banner with dark navy gradient and serif name display
import { useQuery } from '@tanstack/react-query';
import { Settings } from 'lucide-react';
import { api } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';
import { Button } from '../ui/Button';
import type { UserProfile } from '../../types/api';

interface ProfileHeaderProps {
  onEditClick: () => void;
}

export function ProfileHeader({ onEditClick }: ProfileHeaderProps) {
  const { user } = useAuth();

  const { data: profile } = useQuery({
    queryKey: ['profile'],
    queryFn: () => api.get<UserProfile>('/api/profile'),
  });

  const displayName = profile?.usrName ?? user?.usrName ?? '';
  const displayFn = profile?.usrFn ?? '';
  const photoUrl = profile?.usrPhoto;

  return (
    <>
      {/* Dark navy gradient banner */}
      <div className="relative h-32 bg-gradient-to-br from-hero-from via-hero-via to-hero-to overflow-hidden">
        {/* Ambient glow */}
        <div className="absolute top-0 right-0 w-48 h-48 rounded-full bg-white/5 -translate-y-1/2 translate-x-1/2 blur-3xl pointer-events-none" />
        <div className="absolute -bottom-12 left-4">
          {photoUrl ? (
            <img
              src={photoUrl}
              alt={displayName}
              className="h-24 w-24 rounded-full ring-4 ring-surface shadow-md object-cover"
            />
          ) : (
            <div className="h-24 w-24 rounded-full ring-4 ring-surface bg-gradient-to-br from-primary-light to-border shadow-md flex items-center justify-center text-2xl font-bold text-primary font-serif">
              {displayName.charAt(0)}
            </div>
          )}
        </div>
      </div>

      <div className="px-4 pt-14 space-y-1">
        <h1 className="text-xl font-bold text-text-primary font-serif">{displayName}</h1>
        {displayFn && <p className="text-text-tertiary text-sm">{displayFn}기</p>}
      </div>

      <div className="px-4">
        <div className="flex gap-2">
          <Button variant="outline" className="flex-1" onClick={onEditClick}>
            프로필 수정
          </Button>
          <Button variant="ghost" size="icon" className="bg-background">
            <Settings size={20} />
          </Button>
        </div>
      </div>
    </>
  );
}
