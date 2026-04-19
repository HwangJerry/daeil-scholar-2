// AccountLinkForm — Links Kakao to existing member or registers as new, using shared ProfileFieldsSection.
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiClientError } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';
import { ProfileFieldsSection, defaultProfileFieldValues } from './ProfileFieldsSection';
import type { ProfileFieldValues } from './ProfileFieldsSection';
import { useCheckPhone } from '../../hooks/useCheckPhone';
import { useCheckEmail } from '../../hooks/useCheckEmail';
import { Button } from '../ui/Button';
import type { SocialLinkRequest } from '../../types/api';

interface AccountLinkFormProps {
  token: string;
}

export function AccountLinkForm({ token }: AccountLinkFormProps) {
  const navigate = useNavigate();
  const fetchUser = useAuth((s) => s.fetchUser);

  const [profile, setProfile] = useState<ProfileFieldValues>(defaultProfileFieldValues);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const phoneCheck = useCheckPhone(profile.phone);
  const emailCheck = useCheckEmail(profile.email);

  const handleProfileChange = <K extends keyof ProfileFieldValues>(
    key: K,
    value: ProfileFieldValues[K],
  ) => {
    setProfile((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    const body: SocialLinkRequest = {
      token,
      name: profile.name,
      phone: profile.phone,
      email: profile.email,
      fn: profile.fn,
      fmDept: profile.fmDept,
      jobCat: profile.jobCat,
      bizName: profile.bizName,
      bizDesc: profile.bizDesc,
      bizAddr: profile.bizAddr,
      position: profile.position,
      tags: profile.tags,
      usrPhonePublic: profile.usrPhonePublic,
      usrEmailPublic: profile.usrEmailPublic,
    };

    try {
      await api.post('/api/auth/social/link', body);
      await fetchUser();
      navigate('/', { replace: true });
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('처리에 실패했습니다. 다시 시도해주세요.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <ProfileFieldsSection
        values={profile}
        onChange={handleProfileChange}
        phoneCheck={phoneCheck}
        emailCheck={emailCheck}
      />

      {error && <p className="text-sm text-error-text">{error}</p>}

      <Button type="submit" disabled={submitting} className="w-full">
        {submitting ? '처리 중...' : '확인'}
      </Button>
    </form>
  );
}
