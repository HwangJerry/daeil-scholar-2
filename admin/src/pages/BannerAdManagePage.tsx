import { Pencil, Plus, Trash2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { Badge } from '../components/ui/Badge.tsx';
import { Button } from '../components/ui/Button.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { useConfirmDialog } from '../hooks/useConfirmDialog.ts';
import { useBannerAdList } from '../hooks/useBannerAdList.ts';
import type { AdminBannerAdListItem } from '../types/api.ts';

const BANNER_PERIOD_FORMATTER = new Intl.DateTimeFormat('ko-KR', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  timeZone: 'Asia/Seoul',
});

function formatPeriod(startDate: string | null, endDate: string | null): string {
  if (!startDate && !endDate) return '상시';

  const formatDate = (value: string | null) => {
    if (!value) return '제한 없음';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return BANNER_PERIOD_FORMATTER.format(date);
  };

  return `${formatDate(startDate)} ~ ${formatDate(endDate)}`;
}

export function BannerAdManagePage() {
  const navigate = useNavigate();
  const { data: bannerAds, isLoading, isError, refetch, deleteAd, isDeleting } = useBannerAdList();
  const deleteDialog = useConfirmDialog<number>();

  const handleDeleteConfirm = () => {
    const seq = deleteDialog.confirm();
    if (seq != null) deleteAd(seq);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-dark-slate">배너광고 관리</h2>
        <Button onClick={() => navigate('/banner-ad/new')}>
          <Plus className="mr-1 h-4 w-4" />
          새 배너 광고 추가
        </Button>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="w-24 px-4 py-3 font-medium">대표 이미지</th>
              <th className="px-4 py-3 font-medium">배너명</th>
              <th className="px-4 py-3 font-medium">URL</th>
              <th className="w-20 px-4 py-3 text-center font-medium">상태</th>
              <th className="w-16 px-4 py-3 text-center font-medium">순서</th>
              <th className="w-56 px-4 py-3 font-medium">게시기간</th>
              <th className="w-20 px-4 py-3 text-center font-medium">노출수</th>
              <th className="w-20 px-4 py-3 text-center font-medium">클릭수</th>
              <th className="w-24 px-4 py-3 text-center font-medium">작업</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {isError ? (
              <ErrorState colSpan={9} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr>
                <td colSpan={9} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td>
              </tr>
            ) : bannerAds?.length ? (
              bannerAds.map((bannerAd) => (
                <BannerAdTableRow
                  key={bannerAd.bnSeq}
                  bannerAd={bannerAd}
                  onEdit={() => navigate(`/banner-ad/${bannerAd.bnSeq}/edit`)}
                  onDelete={() => deleteDialog.open(bannerAd.bnSeq)}
                />
              ))
            ) : (
              <tr>
                <td colSpan={9} className="px-4 py-8 text-center text-cool-gray">등록된 배너 광고가 없습니다.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <ConfirmDialog
        open={deleteDialog.isOpen}
        onOpenChange={(open) => {
          if (!open) deleteDialog.close();
        }}
        title="배너 광고 삭제"
        description="이 배너 광고를 삭제하시겠습니까?"
        confirmLabel="삭제"
        variant="destructive"
        onConfirm={handleDeleteConfirm}
        isPending={isDeleting}
      />
    </div>
  );
}

function BannerAdTableRow({
  bannerAd,
  onEdit,
  onDelete,
}: {
  bannerAd: AdminBannerAdListItem;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const thumbnailUrl = [...bannerAd.images].sort((left, right) => left.sortOrder - right.sortOrder)[0]?.imageUrl;

  return (
    <tr className="border-b border-border-light hover:bg-background">
      <td className="px-4 py-3">
        {thumbnailUrl ? (
          <img
            src={thumbnailUrl}
            alt={`${bannerAd.bnName} 대표 이미지`}
            className="h-14 w-20 rounded-lg border border-border-light object-cover"
          />
        ) : (
          <div className="flex h-14 w-20 items-center justify-center rounded-lg border border-dashed border-border-light bg-background text-xs text-cool-gray">
            없음
          </div>
        )}
      </td>
      <td className="px-4 py-3 text-dark-slate">{bannerAd.bnName}</td>
      <td className="max-w-xs px-4 py-3 text-cool-gray">
        <p className="truncate">{bannerAd.bnUrl}</p>
      </td>
      <td className="px-4 py-3 text-center">
        <Badge variant={bannerAd.openYn === 'Y' ? 'success' : 'muted'}>
          {bannerAd.openYn === 'Y' ? '활성' : '비활성'}
        </Badge>
      </td>
      <td className="px-4 py-3 text-center text-cool-gray">{bannerAd.indx}</td>
      <td className="px-4 py-3 text-cool-gray">{formatPeriod(bannerAd.bnStartDate, bannerAd.bnEndDate)}</td>
      <td className="px-4 py-3 text-center text-cool-gray">
        {bannerAd.viewCount.toLocaleString()}
      </td>
      <td className="px-4 py-3 text-center text-cool-gray">
        {bannerAd.clickCount.toLocaleString()}
      </td>
      <td className="px-4 py-3 text-center">
        <div className="flex items-center justify-center gap-1">
          <Button variant="ghost" size="icon" onClick={onEdit} aria-label="배너 광고 수정">
            <Pencil className="h-4 w-4 text-cool-gray" />
          </Button>
          <Button variant="ghost" size="icon" onClick={onDelete} aria-label="배너 광고 삭제">
            <Trash2 className="h-4 w-4 text-cool-gray hover:text-error-text" />
          </Button>
        </div>
      </td>
    </tr>
  );
}
