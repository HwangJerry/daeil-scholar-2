// ProfileHeader — User profile header with inline avatar layout, enriched info (company, dept, bio, tags)
import { useQuery } from '@tanstack/react-query';
import { api } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';
import { Button } from '../ui/Button';
import { Bone } from '../ui/Skeleton';
import type { UserProfile } from '../../types/api';

interface ProfileHeaderProps {
  onEditClick: () => void;
}

function ProfileHeaderSkeleton() {
  return (
    <div className="px-4 py-5 flex items-start justify-between gap-4">
      <div className="flex items-start gap-4">
        <Bone className="h-16 w-16 rounded-full flex-shrink-0" />
        <div className="flex-1 space-y-2 pt-1">
          <Bone className="h-5 w-28" />
          <Bone className="h-4 w-36" />
          <Bone className="h-4 w-24" />
        </div>
      </div>
      <Bone className="h-8 w-20 rounded-lg flex-shrink-0" />
    </div>
  );
}

export function ProfileHeader({ onEditClick }: ProfileHeaderProps) {
  const { user } = useAuth();

  const { data: profile, isLoading } = useQuery({
    queryKey: ['profile'],
    queryFn: () => api.get<UserProfile>('/api/profile'),
  });

  if (isLoading) return <ProfileHeaderSkeleton />;

  const displayName = profile?.usrName ?? user?.usrName ?? '';
  const displayFn = profile?.usrFn ?? '';
  const photoUrl = profile?.usrPhoto;
  const jobLine = [profile?.bizName, profile?.position].filter(Boolean).join(' · ');
  const deptLine = [profile?.jobCatName, profile?.bizAddr].filter(Boolean).join(' · ');
  const hasBio = !!profile?.bizDesc;
  const hasTags = (profile?.tags?.length ?? 0) > 0;

  return (
    <div className="px-4 py-5 flex items-start justify-between gap-4">
      {/* Avatar + Info */}
      <div className="flex items-start gap-4">
        {/* Avatar */}
        <div className="flex-shrink-0">
          {photoUrl ? (
            <img
              src={photoUrl}
              alt={displayName}
              className="h-16 w-16 rounded-full ring-2 ring-border shadow-md object-cover"
            />
          ) : (
            <div className="h-16 w-16 rounded-full ring-2 ring-border bg-gradient-to-br from-primary-light to-border shadow-md flex items-center justify-center text-xl font-bold text-primary font-serif">
              {displayName.charAt(0)}
            </div>
          )}
        </div>

        {/* Info block */}
        <div className="flex-1 space-y-1 pt-1">
          {/* Name + generation + dept */}
          <div className="flex items-center gap-2 flex-wrap">
            <h1 className="text-xl font-bold text-text-primary font-serif">{displayName}</h1>
            {(displayFn || profile?.fmDept) && (
              <span className="text-sm text-text-tertiary">
                {[displayFn && `${displayFn}기`, profile?.fmDept].filter(Boolean).join(' ')}
              </span>
            )}
          </div>

          {/* Company · Job type */}
          {jobLine && <p className="text-text-secondary text-sm">{jobLine}</p>}

          {/* Dept · Location */}
          {deptLine && <p className="text-text-tertiary text-xs">{deptLine}</p>}

          {/* Bio */}
          {hasBio && (
            <p className="text-text-secondary text-sm leading-relaxed pt-1">{profile!.bizDesc}</p>
          )}

          {/* Tags */}
          {hasTags && (
            <div className="flex flex-wrap gap-1.5 pt-1">
              {profile!.tags.map((tag) => (
                <span
                  key={tag}
                  className="rounded-full bg-background border border-border text-text-tertiary text-xs px-2.5 py-0.5"
                >
                  #{tag}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Edit button */}
      <Button variant="default" size="sm" onClick={onEditClick} className="flex-shrink-0">
        프로필 수정
      </Button>
    </div>
  );
}
