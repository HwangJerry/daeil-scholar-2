// ProfileFieldsSection — Shared profile input fields used by both RegisterForm and AccountLinkForm.
import { useState } from 'react';
import { X } from 'lucide-react';
import { usePublicJobCategories } from '../../hooks/usePublicJobCategories';
import type { FieldCheckStatus } from '../../hooks/useFieldAvailabilityCheck';
import { Input } from '../ui/Input';
import { DEPARTMENTS } from '../../constants/departments';

const MAX_TAGS = 5;

const selectTextareaClass =
  'w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-sm text-text-secondary outline-none transition-shadow duration-150 focus:ring-2 focus:ring-primary/15 focus:border-primary/30';

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
      <span className="text-xs text-text-placeholder">{isPublic ? '공개' : '비공개'}</span>
    </label>
  );
}

function FieldCheckMessage({ status, label }: { status: FieldCheckStatus; label: string }) {
  if (status === 'idle') return null;
  if (status === 'checking') return <p className="mt-1 text-xs text-text-placeholder">확인 중...</p>;
  if (status === 'available') return <p className="mt-1 text-xs text-success-text">사용 가능한 {label}입니다.</p>;
  if (status === 'unavailable') return <p className="mt-1 text-xs text-error-text">이미 등록된 {label}입니다.</p>;
  return <p className="mt-1 text-xs text-error-text">확인에 실패했습니다. 다시 시도해주세요.</p>;
}

export interface ProfileFieldValues {
  name: string;
  phone: string;
  email: string;
  fn: string;
  fmDept: string;
  jobCat: number | null;
  bizName: string;
  bizDesc: string;
  bizAddr: string;
  position: string;
  tags: string[];
  usrPhonePublic: 'Y' | 'N';
  usrEmailPublic: 'Y' | 'N';
}

export const defaultProfileFieldValues: ProfileFieldValues = {
  name: '',
  phone: '',
  email: '',
  fn: '',
  fmDept: '',
  jobCat: null,
  bizName: '',
  bizDesc: '',
  bizAddr: '',
  position: '',
  tags: [],
  usrPhonePublic: 'N',
  usrEmailPublic: 'N',
};

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
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">이름 *</label>
        <Input
          type="text"
          value={values.name}
          onChange={(e) => onChange('name', e.target.value)}
          required
          placeholder="실명을 입력하세요"
          autoComplete="name"
          disabled={isDisabled('name')}
        />
      </div>
      <div>
        <div className="flex items-center justify-between mb-1">
          <label className="text-sm font-medium text-text-muted">전화번호 *</label>
          <PrivacyToggle
            isPublic={values.usrPhonePublic === 'Y'}
            onToggle={(v) => onChange('usrPhonePublic', v ? 'Y' : 'N')}
          />
        </div>
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
      </div>
      <div>
        <div className="flex items-center justify-between mb-1">
          <label className="text-sm font-medium text-text-muted">이메일 *</label>
          <PrivacyToggle
            isPublic={values.usrEmailPublic === 'Y'}
            onToggle={(v) => onChange('usrEmailPublic', v ? 'Y' : 'N')}
          />
        </div>
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
      </div>

      <div className="pt-1">
        <div className="space-y-3">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">대일외고 기수 *</label>
            <Input
              type="text"
              inputMode="numeric"
              value={values.fn}
              onChange={(e) => onChange('fn', e.target.value.replace(/\D/g, ''))}
              required
              placeholder="숫자만 입력 (예: 10)"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">대일외고 학과 *</label>
            <select
              value={values.fmDept}
              onChange={(e) => onChange('fmDept', e.target.value)}
              required
              className={selectTextareaClass}
            >
              <option value="" disabled>학과를 선택하세요</option>
              {DEPARTMENTS.map((dept) => (
                <option key={dept} value={dept}>
                  {dept}
                </option>
              ))}
            </select>
          </div>
          <div className="border-t border-border pt-3">
            <p className="mb-3 text-xs text-text-placeholder">
              아래 정보는 선택사항입니다. 가입 후 프로필 편집에서도 입력하실 수 있습니다.
            </p>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              업종 <span className="text-text-placeholder">(선택)</span>
            </label>
            <select
              value={values.jobCat ?? ''}
              onChange={(e) => onChange('jobCat', e.target.value ? Number(e.target.value) : null)}
              className={selectTextareaClass}
            >
              <option value="">선택 안함</option>
              {jobCategories.map((cat) => (
                <option key={cat.seq} value={cat.seq}>
                  {cat.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              소속 <span className="text-text-placeholder">(선택)</span>
            </label>
            <Input
              type="text"
              value={values.bizName}
              onChange={(e) => onChange('bizName', e.target.value)}
              placeholder="예: 강남제일부동산"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              근무지 <span className="text-text-placeholder">(선택)</span>
            </label>
            <Input
              type="text"
              value={values.bizAddr}
              onChange={(e) => onChange('bizAddr', e.target.value)}
              placeholder="예: 서울시 강남구 삼성동"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              직책 <span className="text-text-placeholder">(선택)</span>
            </label>
            <Input
              type="text"
              value={values.position}
              onChange={(e) => onChange('position', e.target.value)}
              placeholder="예: 대표, 이사, 팀장"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              소개글 <span className="text-text-placeholder">(선택)</span>
            </label>
            <textarea
              value={values.bizDesc}
              onChange={(e) => onChange('bizDesc', e.target.value)}
              className={selectTextareaClass}
              placeholder="간단한 소개글 (200자 이내)"
              maxLength={200}
              rows={3}
              style={{ resize: 'none' }}
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              태그 <span className="text-text-placeholder">(선택, 최대 {MAX_TAGS}개)</span>
            </label>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {values.tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded-full bg-primary-light text-primary"
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
                  placeholder="태그 입력 후 Enter (공백 불가)"
                />
                {tagError && <p className="mt-1 text-xs text-error-text">{tagError}</p>}
              </>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
