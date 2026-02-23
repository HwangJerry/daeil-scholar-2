// AdManagePage — lists ads with tier badges, edit/delete actions, and view/click stats
import { Trash2, Pencil, Plus } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { Button } from '../components/ui/Button.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { useAdList } from '../hooks/useAdList.ts';
import { useAdStats } from '../hooks/useAdStats.ts';
import { useConfirmDialog } from '../hooks/useConfirmDialog.ts';

const TIER_VARIANT = { PREMIUM: 'premium', GOLD: 'gold', NORMAL: 'muted' } as const;

export function AdManagePage() {
  const navigate = useNavigate();
  const { data: ads, isLoading, isError, refetch, deleteAd } = useAdList();
  const { data: stats } = useAdStats();
  const deleteDialog = useConfirmDialog<number>();

  const handleDeleteConfirm = () => {
    const seq = deleteDialog.confirm();
    if (seq != null) deleteAd(seq);
  };

  const getStats = (maSeq: number) => stats?.find((s) => s.maSeq === maSeq);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-dark-slate">광고 관리</h2>
        <Button onClick={() => navigate('/ad/new')}>
          <Plus className="mr-1 h-4 w-4" />
          새 광고 추가
        </Button>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="px-4 py-3 font-medium">광고명</th>
              <th className="px-4 py-3 font-medium w-24">등급</th>
              <th className="px-4 py-3 font-medium w-20 text-center">상태</th>
              <th className="px-4 py-3 font-medium w-16 text-center">순서</th>
              <th className="px-4 py-3 font-medium w-20 text-center">노출수</th>
              <th className="px-4 py-3 font-medium w-20 text-center">클릭수</th>
              <th className="px-4 py-3 font-medium w-24 text-center">작업</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {isError ? (
              <ErrorState colSpan={7} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
            ) : ads?.length ? (
              ads.map((ad) => {
                const adStats = getStats(ad.maSeq);
                return (
                  <tr key={ad.maSeq} className="border-b border-border-light hover:bg-background">
                    <td className="px-4 py-3 text-dark-slate">{ad.maName ?? '—'}</td>
                    <td className="px-4 py-3">
                      <Badge variant={TIER_VARIANT[ad.adTier] ?? 'muted'}>{ad.adTier}</Badge>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Badge variant={ad.maStatus === 'Y' ? 'success' : 'muted'}>
                        {ad.maStatus === 'Y' ? '활성' : '비활성'}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-center text-cool-gray">{ad.maIndx}</td>
                    <td className="px-4 py-3 text-center text-cool-gray">
                      {adStats != null ? adStats.viewCount.toLocaleString() : '—'}
                    </td>
                    <td className="px-4 py-3 text-center text-cool-gray">
                      {adStats != null ? adStats.clickCount.toLocaleString() : '—'}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <div className="flex items-center justify-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => navigate(`/ad/${ad.maSeq}/edit`)}
                          aria-label="광고 수정"
                        >
                          <Pencil className="h-4 w-4 text-cool-gray" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => deleteDialog.open(ad.maSeq)}
                          aria-label="광고 삭제"
                        >
                          <Trash2 className="h-4 w-4 text-cool-gray hover:text-error-text" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                );
              })
            ) : (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-cool-gray">등록된 광고가 없습니다.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <ConfirmDialog
        open={deleteDialog.isOpen}
        onOpenChange={(open) => { if (!open) deleteDialog.close(); }}
        title="광고 삭제"
        description="이 광고를 삭제하시겠습니까?"
        confirmLabel="삭제"
        variant="destructive"
        onConfirm={handleDeleteConfirm}
      />
    </div>
  );
}
