import { ArrowLeft } from 'lucide-react';
import { useNavigate, useParams } from 'react-router-dom';
import { BannerImageManager } from '../components/banner/BannerImageManager.tsx';
import { Button } from '../components/ui/Button.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { Input } from '../components/ui/Input.tsx';
import { Select } from '../components/ui/Select.tsx';
import { ApiClientError } from '../api/client.ts';
import { useBannerAdDetail } from '../hooks/useBannerAdDetail.ts';
import { useBannerAdForm } from '../hooks/useBannerAdForm.ts';
import { useBannerAdMutations } from '../hooks/useBannerAdMutations.ts';
import type { AdminBannerAdRow, RFC3339UTCDateTime } from '../types/api.ts';

const KST_UTC_OFFSET = '+09:00';

function datetimeLocalToUTC(kst: string): RFC3339UTCDateTime | undefined {
  if (!kst) return undefined;
  return new Date(`${kst}${KST_UTC_OFFSET}`).toISOString() as RFC3339UTCDateTime;
}

function parseBannerAdSeq(value: string | undefined): number | undefined {
  if (value == null) return undefined;

  const parsedSeq = Number(value);
  const isValidSeq = Number.isInteger(parsedSeq) && parsedSeq > 0;
  return isValidSeq ? parsedSeq : undefined;
}

export function BannerAdEditPage() {
  const { bnSeq: routeBnSeq } = useParams<{ bnSeq: string }>();
  const isEditing = routeBnSeq != null;
  const bnSeq = parseBannerAdSeq(routeBnSeq);
  const detail = useBannerAdDetail(bnSeq);

  if (!isEditing) {
    return <BannerAdEditForm bnSeq={undefined} bannerAd={undefined} />;
  }

  if (bnSeq == null) {
    return <BannerAdEditStatus type="not-found" />;
  }

  if (detail.isLoading) {
    return <BannerAdEditStatus type="loading" />;
  }

  if (detail.isError) {
    const isNotFound = detail.error instanceof ApiClientError && detail.error.status === 404;
    if (isNotFound) {
      return <BannerAdEditStatus type="not-found" />;
    }

    return <BannerAdEditStatus type="error" onRetry={() => void detail.refetch()} />;
  }

  if (!detail.data) {
    return <BannerAdEditStatus type="not-found" />;
  }

  return (
    <BannerAdEditForm
      key={detail.data.bnSeq}
      bnSeq={bnSeq}
      bannerAd={detail.data}
    />
  );
}

function BannerAdEditHeader({ isEditing }: { isEditing: boolean }) {
  const navigate = useNavigate();

  return (
    <div className="flex items-center gap-3">
      <Button variant="ghost" size="icon" onClick={() => navigate('/banner-ad')}>
        <ArrowLeft className="h-5 w-5" />
      </Button>
      <h2 className="text-xl font-bold text-dark-slate">
        {isEditing ? '배너 광고 수정' : '배너 광고 등록'}
      </h2>
    </div>
  );
}

function BannerAdEditStatus({
  type,
  onRetry,
}: {
  type: 'loading' | 'not-found' | 'error';
  onRetry?: () => void;
}) {
  let content;
  if (type === 'loading') {
    content = <div className="py-8 text-center text-cool-gray">로딩 중...</div>;
  } else if (type === 'not-found') {
    content = <div className="py-8 text-center text-cool-gray">배너 광고를 찾을 수 없습니다.</div>;
  } else {
    content = <ErrorState message="배너 광고를 불러오는 데 실패했습니다." onRetry={onRetry} />;
  }

  return (
    <div className="space-y-4">
      <BannerAdEditHeader isEditing />
      <div className="rounded-2xl border border-border-light bg-white p-6 shadow-sm">
        {content}
      </div>
    </div>
  );
}

function BannerAdEditForm({
  bnSeq,
  bannerAd,
}: {
  bnSeq: number | undefined;
  bannerAd: AdminBannerAdRow | undefined;
}) {
  const navigate = useNavigate();
  const form = useBannerAdForm(bannerAd);
  const { save, isSaving } = useBannerAdMutations(bnSeq);

  return (
    <div className="space-y-4">
      <BannerAdEditHeader isEditing={bnSeq != null} />

      <div className="space-y-4 rounded-2xl border border-border-light bg-white p-6 shadow-sm">
        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">배너명 *</label>
          <Input
            placeholder="배너명을 입력하세요"
            value={form.bnName}
            onChange={(event) => form.setBnName(event.target.value)}
          />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">링크 URL *</label>
          <Input
            placeholder="https://example.com"
            value={form.bnUrl}
            onChange={(event) => form.setBnUrl(event.target.value)}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="text-sm font-medium text-dark-slate">상태</label>
            <Select
              value={form.openYn}
              onChange={(event) => form.setOpenYn(event.target.value === 'Y' ? 'Y' : 'N')}
            >
              <option value="Y">활성</option>
              <option value="N">비활성</option>
            </Select>
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium text-dark-slate">순서</label>
            <Input
              type="number"
              value={String(form.indx)}
              onChange={(event) => form.setIndx(Number(event.target.value))}
            />
          </div>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">게시 기간 (KST, 비워두면 기간 제한 없음)</label>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs text-cool-gray">시작일시</label>
              <Input
                type="datetime-local"
                value={form.bnStartDate}
                onChange={(event) => form.setBnStartDate(event.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-cool-gray">종료일시</label>
              <Input
                type="datetime-local"
                value={form.bnEndDate}
                onChange={(event) => form.setBnEndDate(event.target.value)}
              />
            </div>
          </div>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-dark-slate">이미지 *</label>
          <BannerImageManager imageUrls={form.imageUrls} onChange={form.setImageUrls} />
        </div>

        <div className="flex justify-end gap-3">
          <Button variant="outline" onClick={() => navigate('/banner-ad')}>
            취소
          </Button>
          <Button
            onClick={() =>
              save({
                bnName: form.bnName,
                bnUrl: form.bnUrl,
                openYn: form.openYn,
                indx: form.indx,
                bnStartDate: datetimeLocalToUTC(form.bnStartDate),
                bnEndDate: datetimeLocalToUTC(form.bnEndDate),
                imageUrls: form.imageUrls,
              })
            }
            disabled={isSaving || !form.isValid}
          >
            {isSaving ? '저장 중...' : '저장'}
          </Button>
        </div>
      </div>
    </div>
  );
}
