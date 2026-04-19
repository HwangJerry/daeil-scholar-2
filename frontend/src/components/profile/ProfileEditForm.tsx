// ProfileEditForm — Edit form for profile info including privacy toggles and image uploads
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';
import { api } from '../../api/client';
import { Button } from '../ui/Button';
import { ImageUploadSection } from './ImageUploadSection';
import type { UserProfile, ProfileUpdateRequest, AlumniFilters } from '../../types/api';

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
      <span className="text-xs text-text-tertiary">{isPublic ? '공개' : '비공개'}</span>
    </label>
  );
}

interface ProfileEditFormProps {
  onSuccess?: () => void;
}

export function ProfileEditForm({ onSuccess }: ProfileEditFormProps) {
  const queryClient = useQueryClient();

  const { data: profile, isLoading } = useQuery({
    queryKey: ['profile'],
    queryFn: () => api.get<UserProfile>('/api/profile'),
  });

  const { data: filters } = useQuery({
    queryKey: ['alumni', 'filters'],
    queryFn: () => api.get<AlumniFilters>('/api/alumni/filters'),
    staleTime: 60 * 60_000,
  });

  const [form, setForm] = useState<ProfileUpdateRequest | null>(null);
  const [tagInput, setTagInput] = useState('');

  const displayForm: ProfileUpdateRequest = form ?? {
    usrName: profile?.usrName ?? '',
    usrFn: profile?.usrFn ?? '',
    usrPhone: profile?.usrPhone ?? '',
    usrEmail: profile?.usrEmail ?? '',
    bizName: profile?.bizName ?? '',
    bizDesc: profile?.bizDesc ?? '',
    bizAddr: profile?.bizAddr ?? '',
    position: profile?.position ?? '',
    fmDept: profile?.fmDept ?? '',
    jobCat: profile?.jobCat ?? null,
    tags: profile?.tags ?? [],
    usrPhonePublic: profile?.usrPhonePublic ?? 'Y',
    usrEmailPublic: profile?.usrEmailPublic ?? 'Y',
  };

  const mutation = useMutation({
    mutationFn: (data: ProfileUpdateRequest) => api.put('/api/profile', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] });
      onSuccess?.();
    },
  });

  const handleChange = (field: keyof ProfileUpdateRequest, value: string | number | null) => {
    setForm((prev) => ({ ...displayForm, ...prev, [field]: value }));
  };

  const handleAddTag = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return;
    if (e.nativeEvent.isComposing) return;
    e.preventDefault();
    const tag = tagInput.trim();
    if (!tag || displayForm.tags.length >= MAX_TAGS) return;
    if (displayForm.tags.includes(tag)) {
      setTagInput('');
      return;
    }
    const newTags = [...displayForm.tags, tag];
    setForm((prev) => ({ ...displayForm, ...prev, tags: newTags }));
    setTagInput('');
  };

  const handleRemoveTag = (tagToRemove: string) => {
    const newTags = displayForm.tags.filter((t) => t !== tagToRemove);
    setForm((prev) => ({ ...displayForm, ...prev, tags: newTags }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate(displayForm);
  };

  if (isLoading) {
    return (
      <div className="space-y-4 rounded-[20px] bg-surface p-6 shadow-card border border-border">
        <div className="h-4 w-32 rounded skeleton-shimmer" />
        <div className="h-9 rounded skeleton-shimmer" />
        <div className="h-9 rounded skeleton-shimmer" />
        <div className="h-9 rounded skeleton-shimmer" />
      </div>
    );
  }

  if (!profile) return null;

  const inputClass =
    'w-full rounded-xl border border-border bg-background px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/15 focus:border-primary/30 transition-shadow duration-150';
  const labelClass = 'text-[13px] font-medium text-text-tertiary';

  return (
    <form onSubmit={handleSubmit} className="rounded-[20px] bg-surface p-6 shadow-card border border-border">
      <h2 className="mb-4 text-lg font-semibold text-text-primary font-serif">프로필 수정</h2>

      {/* Profile photo upload */}
      <ImageUploadSection
        label="프로필 사진"
        currentUrl={profile.usrPhoto}
        uploadUrl="/api/profile/photo"
        onUploaded={() => queryClient.invalidateQueries({ queryKey: ['profile'] })}
        shape="square"
      />

      {/* Name */}
      <div className="mb-4">
        <label className={`mb-1 block ${labelClass}`}>이름</label>
        <input
          type="text"
          value={displayForm.usrName}
          onChange={(e) => handleChange('usrName', e.target.value)}
          className={inputClass}
        />
      </div>

      {/* Class year */}
      <div className="mb-4">
        <label className={`mb-1 block ${labelClass}`}>대일외고 기수</label>
        <input
          type="text"
          value={displayForm.usrFn}
          onChange={(e) => handleChange('usrFn', e.target.value.replace(/\D/g, ''))}
          placeholder="예: 10"
          className={inputClass}
        />
      </div>

      {/* Department */}
      <div className="mb-4">
        <label className={`mb-1 block ${labelClass}`}>대일외고 학과</label>
        <input
          type="text"
          value={displayForm.fmDept}
          onChange={(e) => handleChange('fmDept', e.target.value)}
          placeholder="예: 독일어과"
          className={inputClass}
        />
      </div>

      {/* Phone with privacy toggle */}
      <div className="mb-4">
        <div className="flex items-center justify-between mb-1">
          <label className={labelClass}>연락처</label>
          <PrivacyToggle
            isPublic={displayForm.usrPhonePublic === 'Y'}
            onToggle={(v) => handleChange('usrPhonePublic', v ? 'Y' : 'N')}
          />
        </div>
        <input
          type="tel"
          value={displayForm.usrPhone}
          onChange={(e) => handleChange('usrPhone', e.target.value)}
          className={inputClass}
        />
      </div>

      {/* Email with privacy toggle */}
      <div className="mb-4">
        <div className="flex items-center justify-between mb-1">
          <label className={labelClass}>이메일</label>
          <PrivacyToggle
            isPublic={displayForm.usrEmailPublic === 'Y'}
            onToggle={(v) => handleChange('usrEmailPublic', v ? 'Y' : 'N')}
          />
        </div>
        <input
          type="email"
          value={displayForm.usrEmail}
          onChange={(e) => handleChange('usrEmail', e.target.value)}
          className={inputClass}
        />
      </div>

      {/* Divider */}
      <hr className="my-5 border-border" />
      <h3 className="mb-3 text-base font-semibold text-text-primary font-serif">사업장 정보</h3>

      {/* Biz card upload */}
      <ImageUploadSection
        label="명함 이미지"
        currentUrl={profile.usrBizCard}
        uploadUrl="/api/profile/bizcard"
        onUploaded={() => queryClient.invalidateQueries({ queryKey: ['profile'] })}
        shape="card"
      />

      {/* Job Category */}
      <div className="mb-4">
        <label className={`mb-1 block ${labelClass}`}>업종</label>
        <select
          value={displayForm.jobCat ?? ''}
          onChange={(e) => {
            const val = e.target.value;
            handleChange('jobCat', val ? Number(val) : null);
          }}
          className={cn(inputClass, 'bg-background')}
        >
          <option value="">선택 안함</option>
          {filters?.jobCategories?.map((cat) => (
            <option key={cat.seq} value={cat.seq}>
              {cat.name}
            </option>
          ))}
        </select>
      </div>

      {/* Business Name */}
      <div className="mb-4">
        <label className={`mb-1 block ${labelClass}`}>소속</label>
        <input
          type="text"
          value={displayForm.bizName}
          onChange={(e) => handleChange('bizName', e.target.value)}
          placeholder="예: 강남제일부동산"
          className={inputClass}
        />
      </div>

      {/* Business Address */}
      <div className="mb-4">
        <label className={`mb-1 block ${labelClass}`}>근무지</label>
        <input
          type="text"
          value={displayForm.bizAddr}
          onChange={(e) => handleChange('bizAddr', e.target.value)}
          placeholder="예: 서울시 강남구 삼성동"
          className={inputClass}
        />
      </div>

      {/* Position / 직책 */}
      <div className="mb-4">
        <label className={`mb-1 block ${labelClass}`}>직책</label>
        <input
          type="text"
          value={displayForm.position}
          onChange={(e) => handleChange('position', e.target.value)}
          placeholder="예: 대표, 이사, 팀장"
          className={inputClass}
        />
      </div>

      {/* Business Description */}
      <div className="mb-4">
        <label className={`mb-1 block ${labelClass}`}>소개글</label>
        <textarea
          value={displayForm.bizDesc}
          onChange={(e) => handleChange('bizDesc', e.target.value)}
          placeholder="간단한 소개글 (200자 이내)"
          maxLength={200}
          rows={3}
          className={cn(inputClass, 'resize-none')}
        />
      </div>

      {/* Tags */}
      <div className="mb-6">
        <label className={`mb-1 block ${labelClass}`}>태그 (최대 {MAX_TAGS}개)</label>
        <div className="flex flex-wrap gap-1.5 mb-2">
          {displayForm.tags.map((tag) => (
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
        {displayForm.tags.length < MAX_TAGS && (
          <input
            type="text"
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            onKeyDown={handleAddTag}
            placeholder="태그 입력 후 Enter"
            className={inputClass}
          />
        )}
      </div>

      <Button type="submit" disabled={mutation.isPending} className="w-full">
        {mutation.isPending ? '저장 중...' : '저장'}
      </Button>
      {mutation.isError && (
        <p className="mt-2 text-center text-sm text-error">저장에 실패했습니다. 다시 시도해주세요.</p>
      )}
    </form>
  );
}
