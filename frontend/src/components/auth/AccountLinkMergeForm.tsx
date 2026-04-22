// AccountLinkMergeForm — Integrated signup form for users whose phone matches an existing member. Prefills from the matched member and locks core identifiers.
import { useEffect, useRef, useState } from 'react';
import { ProfileFieldsSection, defaultProfileFieldValues } from './ProfileFieldsSection';
import type { ProfileFieldValues } from './ProfileFieldsSection';
import { useSocialLinkPrefill } from '../../hooks/useSocialLinkPrefill';
import { useSocialLinkPhoneMatch } from '../../hooks/useSocialLinkPhoneMatch';
import { useAccountLinkSubmit } from '../../hooks/useAccountLinkSubmit';
import { KakaoProfileImagePreview } from './KakaoProfileImagePreview';
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
  const didPrefill = useRef(false);

  const prefill = useSocialLinkPrefill(token);
  const mergeLookup = useSocialLinkPhoneMatch({ token, phone: initialPhone });

  useEffect(() => {
    if (didPrefill.current) return;
    const matched = mergeLookup.profile;
    if (!matched) return;
    setProfile({
      name: matched.name,
      phone: initialPhone,
      email: matched.email,
      fn: matched.fn,
      fmDept: matched.fmDept,
      jobCat: matched.jobCat,
      bizName: matched.bizName,
      bizDesc: matched.bizDesc,
      bizAddr: matched.bizAddr,
      position: matched.position,
      tags: matched.tags,
      usrPhonePublic: matched.usrPhonePublic,
      usrEmailPublic: matched.usrEmailPublic,
    });
    didPrefill.current = true;
  }, [mergeLookup.profile, initialPhone]);

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
    });
  };

  const isLoading = !didPrefill.current && mergeLookup.status !== 'unmatched' && mergeLookup.status !== 'error';

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <KakaoProfileImagePreview imageUrl={prefill.data?.profileImageUrl || null} />

      {mergeLookup.status === 'unmatched' && (
        <div className="mb-4 rounded-lg bg-error-subtle px-4 py-3 text-sm text-error-text">
          입력하신 번호와 일치하는 기존 회원을 찾을 수 없습니다. 신규 가입으로 돌아가주세요.
        </div>
      )}

      {isLoading ? (
        <p className="py-8 text-center text-sm text-text-placeholder">회원 정보를 불러오는 중...</p>
      ) : (
        <ProfileFieldsSection
          values={profile}
          onChange={handleProfileChange}
          disabledFields={MERGE_DISABLED_FIELDS}
        />
      )}

      {error && <p className="text-sm text-error-text">{error}</p>}

      <Button type="submit" disabled={submitting || isLoading} className="w-full">
        {submitting ? '처리 중...' : '통합 회원가입 완료'}
      </Button>
    </form>
  );
}
