// AdManagePage — lists ads with tier badges, delete confirmation dialog, and error handling
import { Trash2 } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { useAdList } from '../hooks/useAdList.ts';
import { useConfirmDialog } from '../hooks/useConfirmDialog.ts';

const TIER_VARIANT = { PREMIUM: 'premium', GOLD: 'gold', NORMAL: 'muted' } as const;

export function AdManagePage() {
  const { data: ads, isLoading, isError, refetch, deleteAd } = useAdList();
  const deleteDialog = useConfirmDialog<number>();

  const handleDeleteConfirm = () => {
    const seq = deleteDialog.confirm();
    if (seq != null) deleteAd(seq);
  };

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold text-dark-slate">광고 관리</h2>

      <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="px-4 py-3 font-medium">광고명</th>
              <th className="px-4 py-3 font-medium w-24">등급</th>
              <th className="px-4 py-3 font-medium w-20 text-center">상태</th>
              <th className="px-4 py-3 font-medium w-16 text-center">순서</th>
              <th className="px-4 py-3 font-medium w-20 text-center">작업</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {isError ? (
              <ErrorState colSpan={5} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
            ) : ads?.length ? (
              ads.map((ad) => (
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
                  <td className="px-4 py-3 text-center">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => deleteDialog.open(ad.maSeq)}
                      aria-label="광고 삭제"
                    >
                      <Trash2 className="h-4 w-4 text-cool-gray hover:text-error-text" />
                    </Button>
                  </td>
                </tr>
              ))
            ) : (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">등록된 광고가 없습니다.</td></tr>
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
