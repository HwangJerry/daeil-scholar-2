// Displays the user's profile banner with avatar, name, class info, and edit button
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
      <div className="relative h-28 bg-gradient-to-r from-primary to-violet-500">
        <div className="absolute -bottom-10 left-4">
          {photoUrl ? (
            <img
              src={photoUrl}
              alt={displayName}
              className="h-24 w-24 rounded-full ring-4 ring-surface shadow-md object-cover"
            />
          ) : (
            <div className="h-24 w-24 rounded-full ring-4 ring-surface bg-gradient-to-br from-primary-light to-indigo-100 shadow-md flex items-center justify-center text-2xl font-semibold text-primary">
              {displayName.charAt(0)}
            </div>
          )}
        </div>
      </div>

      <div className="px-4 pt-12 space-y-1">
        <h1 className="text-xl font-bold text-text-primary">{displayName}</h1>
        {displayFn && <p className="text-text-tertiary">{displayFn}기</p>}
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
