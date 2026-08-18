// ProfileFieldsSection — Shared profile input fields used by both RegisterForm and AccountLinkForm.
import { useState } from 'react';
import { X } from 'lucide-react';
import { usePublicJobCategories } from '../../hooks/usePublicJobCategories';
import type { FieldCheckStatus } from '../../hooks/useFieldAvailabilityCheck';
import { Input } from '../ui/Input';
import { DEPARTMENTS } from '../../constants/departments';
import type { ProfileFieldValues } from './profileFieldValues';
import {
  AuthField,
  AuthFieldMessage,
  AuthInlineField,
  AuthSectionText,
  AuthSelect,
  AuthTextarea,
} from './AuthFormPrimitives';

const MAX_TAGS = 5;

function PrivacyToggle({
  isPublic,
  onToggle,
}: {
  isPublic: boolean;
  onToggle: (v: boolean) => void;
}) {
  return (
    <label className="group inline-flex items-center gap-1.5 cursor-pointer">
      <input
        type="checkbox"
        className="sr-only"
        checked={isPublic}
        onChange={(e) => onToggle(e.target.checked)}
      />
      <div className="relative w-9 h-5 rounded-full bg-border group-has-[:checked]:bg-primary transition-colors">
        <span className="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-surface shadow-sm transition-transform group-has-[:checked]:translate-x-4" />
      </div>
      <span className="text-caption text-text-placeholder">{isPublic ? '공개' : '비공개'}</span>
    </label>
  );
}

function FieldCheckMessage({ status, label }: { status: FieldCheckStatus; label: string }) {
  if (status === 'idle') return null;
  if (status === 'checking') return <AuthFieldMessage>확인 중...</AuthFieldMessage>;
  if (status === 'available') {
    return <AuthFieldMessage tone="success">사용 가능한 {label}입니다.</AuthFieldMessage>;
  }
  if (status === 'unavailable') {
    return <AuthFieldMessage tone="error">이미 등록된 {label}입니다.</AuthFieldMessage>;
  }
  return <AuthFieldMessage tone="error">확인에 실패했습니다. 다시 시도해주세요.</AuthFieldMessage>;
}

interface FieldCheckProps {
  status: FieldCheckStatus;
  onBlur: () => void;
}

interface ProfileFieldsSectionProps {
  values: ProfileFieldValues;
  onChange: <K extends keyof ProfileFieldValues>(key: K, value: ProfileFieldValues[K]) => void;
  phoneCheck?: FieldCheckProps;
  emailCheck?: FieldCheckProps;
  /** Field keys that should render as disabled (e.g. merge-mode locks 'name', 'phone', 'email'). */
  disabledFields?: Array<keyof ProfileFieldValues>;
}

export function ProfileFieldsSection({ values, onChange, phoneCheck, emailCheck, disabledFields }: ProfileFieldsSectionProps) {
  const disabledSet = new Set<keyof ProfileFieldValues>(disabledFields ?? []);
  const isDisabled = (k: keyof ProfileFieldValues) => disabledSet.has(k);
  const [tagInput, setTagInput] = useState('');
  const [tagError, setTagError] = useState('');
  const { data: jobCategories = [] } = usePublicJobCategories();

  const handleAddTag = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return;
    if (e.nativeEvent.isComposing) return;
    e.preventDefault();
    const tag = tagInput.trim();
    if (!tag || values.tags.length >= MAX_TAGS) return;
    if (/\s/.test(tag)) {
      setTagError('태그에는 공백을 포함할 수 없습니다');
      return;
    }
    if (values.tags.includes(tag)) {
      setTagInput('');
      setTagError('');
      return;
    }
    onChange('tags', [...values.tags, tag]);
    setTagInput('');
    setTagError('');
  };

  const handleRemoveTag = (tagToRemove: string) => {
    onChange('tags', values.tags.filter((t) => t !== tagToRemove));
  };

  return (
    <>
      <AuthField label="이름 *">
        <Input
          type="text"
          value={values.name}
          onChange={(e) => onChange('name', e.target.value)}
          required
          placeholder="실명을 입력하세요"
          autoComplete="name"
          disabled={isDisabled('name')}
        />
      </AuthField>
      <AuthInlineField
        label="전화번호 *"
        action={(
          <PrivacyToggle
            isPublic={values.usrPhonePublic === 'Y'}
            onToggle={(v) => onChange('usrPhonePublic', v ? 'Y' : 'N')}
          />
        )}
      >
        <Input
          type="tel"
          value={values.phone}
          onChange={(e) => onChange('phone', e.target.value)}
          onBlur={phoneCheck?.onBlur}
          required
          placeholder="010-0000-0000"
          autoComplete="tel"
          disabled={isDisabled('phone')}
        />
        {phoneCheck && !isDisabled('phone') && <FieldCheckMessage status={phoneCheck.status} label="전화번호" />}
      </AuthInlineField>
      <AuthInlineField
        label="이메일 *"
        action={(
          <PrivacyToggle
            isPublic={values.usrEmailPublic === 'Y'}
            onToggle={(v) => onChange('usrEmailPublic', v ? 'Y' : 'N')}
          />
        )}
      >
        <Input
          type="email"
          value={values.email}
          onChange={(e) => onChange('email', e.target.value)}
          onBlur={emailCheck?.onBlur}
          required
          placeholder="example@email.com"
          autoComplete="email"
          disabled={isDisabled('email')}
        />
        {emailCheck && !isDisabled('email') && <FieldCheckMessage status={emailCheck.status} label="이메일" />}
      </AuthInlineField>

      <div className="pt-1">
        <div className="space-y-3">
          <AuthField label="대일외고 기수 *">
            <Input
              type="text"
              inputMode="numeric"
              value={values.fn}
              onChange={(e) => onChange('fn', e.target.value.replace(/\D/g, ''))}
              required
              placeholder="숫자만 입력 (예: 10)"
            />
          </AuthField>
          <AuthField label="대일외고 학과 *">
            <AuthSelect
              value={values.fmDept}
              onChange={(e) => onChange('fmDept', e.target.value)}
              required
            >
              <option value="" disabled>학과를 선택하세요</option>
              {DEPARTMENTS.map((dept) => (
                <option key={dept} value={dept}>
                  {dept}
                </option>
              ))}
            </AuthSelect>
          </AuthField>
          <div className="border-t border-border pt-3">
            <AuthSectionText tone="muted">
              아래 정보는 선택사항입니다. 가입 후 프로필 편집에서도 입력하실 수 있습니다.
            </AuthSectionText>
          </div>
          <AuthField label={<>업종 <span className="text-text-placeholder">(선택)</span></>}>
            <AuthSelect
              value={values.jobCat ?? ''}
              onChange={(e) => onChange('jobCat', e.target.value ? Number(e.target.value) : null)}
            >
              <option value="">선택 안함</option>
              {jobCategories.map((cat) => (
                <option key={cat.seq} value={cat.seq}>
                  {cat.name}
                </option>
              ))}
            </AuthSelect>
          </AuthField>
          <AuthField label={<>소속 <span className="text-text-placeholder">(선택)</span></>}>
            <Input
              type="text"
              value={values.bizName}
              onChange={(e) => onChange('bizName', e.target.value)}
              placeholder="예: 강남제일부동산"
            />
          </AuthField>
          <AuthField label={<>근무지 <span className="text-text-placeholder">(선택)</span></>}>
            <Input
              type="text"
              value={values.bizAddr}
              onChange={(e) => onChange('bizAddr', e.target.value)}
              placeholder="예: 서울시 강남구 삼성동"
            />
          </AuthField>
          <AuthField label={<>직책 <span className="text-text-placeholder">(선택)</span></>}>
            <Input
              type="text"
              value={values.position}
              onChange={(e) => onChange('position', e.target.value)}
              placeholder="예: 대표, 이사, 팀장"
            />
          </AuthField>
          <AuthField label={<>소개글 <span className="text-text-placeholder">(선택)</span></>}>
            <AuthTextarea
              value={values.bizDesc}
              onChange={(e) => onChange('bizDesc', e.target.value)}
              placeholder="간단한 소개글 (200자 이내)"
              maxLength={200}
              rows={3}
            />
          </AuthField>
          <AuthField label={<>태그 <span className="text-text-placeholder">(선택, 최대 {MAX_TAGS}개)</span></>}>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {values.tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 rounded-full bg-primary-light px-2.5 py-1 text-caption font-medium text-primary"
                >
                  #{tag}
                  <button
                    type="button"
                    onClick={() => handleRemoveTag(tag)}
                    className="hover:text-error transition-colors"
                  >
                    <X size={12} />
                  </button>
                </span>
              ))}
            </div>
            {values.tags.length < MAX_TAGS && (
              <>
                <Input
                  type="text"
                  value={tagInput}
                  onChange={(e) => {
                    setTagInput(e.target.value);
                    if (tagError) setTagError('');
                  }}
                  onKeyDown={handleAddTag}
                  placeholder="태그 입력 후 Enter (스페이스 불가)"
                />
                {tagError && <AuthFieldMessage tone="error">{tagError}</AuthFieldMessage>}
              </>
            )}
          </AuthField>
        </div>
      </div>
    </>
  );
}
