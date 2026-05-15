// AccountLinkMergeForm — Integrated signup form for users whose phone matches an existing member. Prefills from the matched member and locks core identifiers.
import { useState } from 'react';
import { ProfileFieldsSection } from './ProfileFieldsSection';
import { defaultProfileFieldValues } from './profileFieldValues';
import type { ProfileFieldValues } from './profileFieldValues';
import { useSocialLinkPrefill } from '../../hooks/useSocialLinkPrefill';
import { useSocialLinkExistingMember } from '../../hooks/useSocialLinkExistingMember';
import { useAccountLinkSubmit } from '../../hooks/useAccountLinkSubmit';
import { SignupProfileImageEditor } from './SignupProfileImageEditor';
import { Button } from '../ui/Button';
import { isValidDepartment } from '../../constants/departments';

const FN_REGEX = /^[0-9]+$/;

interface AccountLinkMergeFormProps {
  token: string;
  initialPhone: string;
}

const MERGE_DISABLED_FIELDS: Array<keyof ProfileFieldValues> = ['name', 'phone', 'email'];

export function AccountLinkMergeForm({ token, initialPhone }: AccountLinkMergeFormProps) {
  const { submitting, error, submit, setError } = useAccountLinkSubmit();

  const [profile, setProfile] = useState<ProfileFieldValues>({
    ...defaultProfileFieldValues,
    phone: initialPhone,
  });
  const [didPrefill, setDidPrefill] = useState(false);
  const [photoOverride, setPhotoOverride] = useState<{ url: string | null } | null>(null);

  const prefill = useSocialLinkPrefill(token);
  const existing = useSocialLinkExistingMember(token, initialPhone);

  // Prefill once during render when the matched profile arrives — React 19 idiom avoids the
  // cascading-effect re-render and is enforced by the lint rule.
  const matched = existing.data?.matched === true && !!existing.data.profile;
  if (!didPrefill && matched) {
    const p = existing.data!.profile!;
    setDidPrefill(true);
    setProfile({
      name: p.name,
      phone: initialPhone,
      email: p.email,
      fn: p.fn,
      fmDept: p.fmDept,
      jobCat: p.jobCat,
      bizName: p.bizName,
      bizDesc: p.bizDesc,
      bizAddr: p.bizAddr,
      position: p.position,
      tags: p.tags,
      usrPhonePublic: p.usrPhonePublic,
      usrEmailPublic: p.usrEmailPublic,
    });
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
      mode: 'merge',
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

  const unmatched = existing.data?.matched === false;
  const showForm = didPrefill;

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <SignupProfileImageEditor token={token} imageUrl={photoUrl} onChange={handlePhotoChange} />

      {unmatched && (
        <div className="mb-4 rounded-lg bg-error-subtle px-4 py-3 text-sm text-error-text">
          입력하신 번호와 일치하는 기존 회원을 찾을 수 없습니다. 신규 가입으로 돌아가주세요.
        </div>
      )}

      {existing.isError && (
        <div className="mb-4 rounded-lg bg-error-subtle px-4 py-3 text-sm text-error-text">
          <p className="mb-2">회원 정보를 불러오지 못했습니다. 잠시 후 다시 시도해주세요.</p>
          <button
            type="button"
            onClick={() => existing.refetch()}
            className="text-xs underline"
          >
            다시 시도
          </button>
        </div>
      )}

      {!showForm && !unmatched && !existing.isError ? (
        <p className="py-8 text-center text-sm text-text-placeholder">회원 정보를 불러오는 중...</p>
      ) : showForm ? (
        <ProfileFieldsSection
          values={profile}
          onChange={handleProfileChange}
          disabledFields={MERGE_DISABLED_FIELDS}
        />
      ) : null}

      {error && <p className="text-sm text-error-text">{error}</p>}

      <Button type="submit" disabled={submitting || !showForm} className="w-full">
        {submitting ? '처리 중...' : '통합 회원가입 완료'}
      </Button>
    </form>
  );
}
