// AccountLinkNewForm — Fresh Kakao signup form. Prefills email/profile-image from Kakao and shows a merge banner when the typed phone matches an existing member.
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ProfileFieldsSection } from './ProfileFieldsSection';
import { defaultProfileFieldValues } from './profileFieldValues';
import type { ProfileFieldValues } from './profileFieldValues';
import { useCheckPhone } from '../../hooks/useCheckPhone';
import { useCheckEmail } from '../../hooks/useCheckEmail';
import { useSocialLinkPrefill } from '../../hooks/useSocialLinkPrefill';
import { useSocialLinkPhoneMatch } from '../../hooks/useSocialLinkPhoneMatch';
import { useAccountLinkSubmit } from '../../hooks/useAccountLinkSubmit';
import { PhoneMatchBanner } from './PhoneMatchBanner';
import { KakaoProfileImagePreview } from './KakaoProfileImagePreview';
import { Button } from '../ui/Button';
import { isValidDepartment } from '../../constants/departments';

const FN_REGEX = /^[0-9]+$/;

interface AccountLinkNewFormProps {
  token: string;
}

export function AccountLinkNewForm({ token }: AccountLinkNewFormProps) {
  const navigate = useNavigate();
  const { submitting, error, submit, setError } = useAccountLinkSubmit();

  const [profile, setProfile] = useState<ProfileFieldValues>(defaultProfileFieldValues);
  const [didPrefillEmail, setDidPrefillEmail] = useState(false);

  const prefill = useSocialLinkPrefill(token);
  const phoneBannerLookup = useSocialLinkPhoneMatch({ token, phone: profile.phone });
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

  const handleConfirmMerge = () => {
    navigate(`/login/link?token=${encodeURIComponent(token)}&mode=merge`, {
      replace: true,
      state: { phone: profile.phone },
    });
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
    });
  };

  const bannerMatch = phoneBannerLookup.status === 'matched' ? phoneBannerLookup.profile : null;

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <KakaoProfileImagePreview imageUrl={prefill.data?.profileImageUrl || null} />

      {bannerMatch && (
        <PhoneMatchBanner
          matchedName={bannerMatch.name}
          matchedFN={bannerMatch.fn}
          onConfirm={handleConfirmMerge}
        />
      )}

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
