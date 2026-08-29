// AccountLinkNewForm — Fresh Kakao signup form with provider email/profile-image prefill.
import { useState } from 'react';
import { ProfileFieldsSection } from './ProfileFieldsSection';
import { defaultProfileFieldValues } from './profileFieldValues';
import type { ProfileFieldValues } from './profileFieldValues';
import { useCheckPhone } from '../../hooks/useCheckPhone';
import { useCheckEmail } from '../../hooks/useCheckEmail';
import { useSocialLinkPrefill } from '../../hooks/useSocialLinkPrefill';
import { useAccountLinkSubmit } from '../../hooks/useAccountLinkSubmit';
import { SignupProfileImageEditor } from './SignupProfileImageEditor';
import { Button } from '../ui/Button';
import { isValidDepartment } from '../../constants/departments';

const FN_REGEX = /^[0-9]+$/;

interface AccountLinkNewFormProps {
  token: string;
}

export function AccountLinkNewForm({ token }: AccountLinkNewFormProps) {
  const { submitting, error, submit, setError } = useAccountLinkSubmit();

  const [profile, setProfile] = useState<ProfileFieldValues>(defaultProfileFieldValues);
  const [didPrefillEmail, setDidPrefillEmail] = useState(false);
  const [photoOverride, setPhotoOverride] = useState<{ url: string | null } | null>(null);

  const prefill = useSocialLinkPrefill(token);
  const phoneCheck = useCheckPhone(profile.phone);
  const emailCheck = useCheckEmail(profile.email);

  // Prefill email from Kakao once during render when it becomes available.
  // setState-during-render is the React 19 idiom; it skips the cascading effect re-render.
  const kakaoEmail = prefill.data?.email;
  if (!didPrefillEmail && kakaoEmail) {
    setDidPrefillEmail(true);
    setProfile((prev) => (prev.email ? prev : { ...prev, email: kakaoEmail }));
  }

  const handleProfileChange = <K extends keyof ProfileFieldValues>(
    key: K,
    value: ProfileFieldValues[K],
  ) => {
    setProfile((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!FN_REGEX.test(profile.fn)) {
      setError('기수는 숫자로 입력해주세요.');
      return;
    }
    if (!isValidDepartment(profile.fmDept)) {
      setError('학과를 선택해주세요.');
      return;
    }
    void submit({
      token,
      mode: 'new',
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
      ...(photoOverride !== null && { profileImageUrl: photoOverride.url ?? '' }),
    });
  };

  const photoUrl =
    photoOverride !== null ? photoOverride.url : prefill.data?.profileImageUrl || null;
  const handlePhotoChange = (url: string | null) => setPhotoOverride({ url });

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <SignupProfileImageEditor token={token} imageUrl={photoUrl} onChange={handlePhotoChange} />

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
