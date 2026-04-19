// AdEditPage — create or edit an advertisement with image upload support
import { useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Input } from '../components/ui/Input.tsx';
import { Select } from '../components/ui/Select.tsx';
import { useAdList } from '../hooks/useAdList.ts';
import { useAdForm } from '../hooks/useAdForm.ts';
import { useAdMutations } from '../hooks/useAdMutations.ts';
import { uploadImage } from '../components/editor/uploadImage.ts';
import type { AdminAdListItem } from '../types/api.ts';

const TIER_OPTIONS = [
  { value: 'PREMIUM', label: '1-tier (가장 핫한 동문 소식)' },
  { value: 'GOLD',    label: '2-tier (추천 동문 소식)' },
  { value: 'NORMAL',  label: '3-tier (동문 소식)' },
] as const;

// datetimeLocalToUTC: "2026-03-17T09:00" (KST local input) → UTC ISO string
function datetimeLocalToUTC(kst: string): string | undefined {
  if (!kst) return undefined;
  return new Date(kst).toISOString();
}

export function AdEditPage() {
  const { maSeq } = useParams<{ maSeq: string }>();
  const { data: ads } = useAdList();

  const seq = maSeq ? Number(maSeq) : undefined;
  const existing = seq != null ? ads?.find((a) => a.maSeq === seq) : undefined;

  return <AdEditForm key={seq ?? 'new'} maSeq={seq} ad={existing} />;
}

function AdEditForm({
  maSeq,
  ad,
}: {
  maSeq: number | undefined;
  ad: AdminAdListItem | undefined;
}) {
  const navigate = useNavigate();
  const form = useAdForm(ad);
  const { save, isSaving } = useAdMutations(maSeq);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const isLimitedTier = form.adTier === 'PREMIUM' || form.adTier === 'GOLD';

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      const url = await uploadImage(file);
      form.setMaImg(url);
    } catch {
      // upload error is handled by the API client; form remains unchanged
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate('/ad')}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h2 className="text-xl font-bold text-dark-slate">
          {maSeq != null ? '광고 수정' : '광고 등록'}
        </h2>
      </div>

      <div className="space-y-4 rounded-2xl border border-border-light bg-white p-6 shadow-sm">
        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">광고명 *</label>
          <Input
            placeholder="광고명을 입력하세요"
            value={form.maName}
            onChange={(e) => form.setMaName(e.target.value)}
          />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">링크 URL *</label>
          <Input
            placeholder="https://example.com"
            value={form.maUrl}
            onChange={(e) => form.setMaUrl(e.target.value)}
          />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">이미지</label>
          <div className="flex gap-2">
            <Input
              placeholder="이미지 URL을 입력하거나 파일을 업로드하세요"
              value={form.maImg}
              onChange={(e) => form.setMaImg(e.target.value)}
              className="flex-1"
            />
            <Button variant="outline" onClick={() => fileInputRef.current?.click()}>
              파일 업로드
            </Button>
          </div>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={(e) => { void handleFileChange(e); }}
          />
          {form.maImg && (
            <img
              src={form.maImg}
              alt="광고 이미지 미리보기"
              className="mt-2 h-24 w-auto rounded-lg border border-border-light object-contain"
            />
          )}
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="text-sm font-medium text-dark-slate">상태</label>
            <Select value={form.maStatus} onChange={(e) => form.setMaStatus(e.target.value)}>
              <option value="Y">활성</option>
              <option value="N">비활성</option>
            </Select>
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium text-dark-slate">등급</label>
            <Select value={form.adTier} onChange={(e) => form.setAdTierAndLabel(e.target.value)}>
              {TIER_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </Select>
            {isLimitedTier && (
              <p className="text-xs text-amber-600">이 등급은 1개만 활성화 가능합니다.</p>
            )}
          </div>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">카드 라벨</label>
          <Input
            placeholder="추천 동문 소식"
            value={form.adTitleLabel}
            onChange={(e) => form.setAdTitleLabel(e.target.value)}
          />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">게시 기간 (KST, 비워두면 기간 제한 없음)</label>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs text-cool-gray">시작일시</label>
              <Input
                type="datetime-local"
                value={form.adStartDate}
                onChange={(e) => form.setAdStartDate(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-cool-gray">종료일시</label>
              <Input
                type="datetime-local"
                value={form.adEndDate}
                onChange={(e) => form.setAdEndDate(e.target.value)}
              />
            </div>
          </div>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">순서</label>
          <Input
            type="number"
            value={String(form.maIndx)}
            onChange={(e) => form.setMaIndx(Number(e.target.value))}
          />
        </div>

        <div className="flex justify-end gap-3">
          <Button variant="outline" onClick={() => navigate('/ad')}>취소</Button>
          <Button
            onClick={() => save({
              maName: form.maName,
              maUrl: form.maUrl,
              maImg: form.maImg,
              maStatus: form.maStatus,
              adTier: form.adTier,
              adTitleLabel: form.adTitleLabel,
              maIndx: form.maIndx,
              adStartDate: datetimeLocalToUTC(form.adStartDate),
              adEndDate: datetimeLocalToUTC(form.adEndDate),
            })}
            disabled={isSaving || !form.isValid}
          >
            {isSaving ? '저장 중...' : '저장'}
          </Button>
        </div>
      </div>
    </div>
  );
}
