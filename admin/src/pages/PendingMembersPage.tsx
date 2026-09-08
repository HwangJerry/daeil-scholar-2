// PendingMembersPage — approval queue for new member registration requests
import { useState } from 'react';
import { Button } from '../components/ui/Button.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { usePendingMembers } from '../hooks/usePendingMembers.ts';
import type { PendingVerification } from '../hooks/usePendingMembers.ts';

type ActionType = 'approve' | 'reject';

interface PendingAction {
  member: PendingVerification;
  type: ActionType;
}

export function PendingMembersPage() {
  const { data, isLoading, isError, refetch, approve, reject, isApproving, isRejecting } =
    usePendingMembers();
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);

  const [rejectionReason, setRejectionReason] = useState('');
  const isRejectingAction = pendingAction?.type === 'reject';
  const isActioning = isApproving || isRejecting;

  const handleConfirm = () => {
    if (!pendingAction) return;
    const { member, type } = pendingAction;
    if (type === 'approve') {
      approve(member, { onSettled: () => setPendingAction(null) });
    } else {
      reject({ member, reason: rejectionReason }, { onSettled: () => setPendingAction(null) });
    }
  };

  const items = data?.items ?? [];

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold text-dark-slate">가입 신청 관리</h2>
      <p className="text-sm text-cool-gray">
        제출된 동문 인증 신청과 재승인 신청을 검토하세요. 승인 후 동문 기능을 이용할 수 있습니다.
      </p>

      <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="px-4 py-3 font-medium">회원번호</th>
              <th className="px-4 py-3 font-medium">이름</th>
              <th className="px-4 py-3 font-medium w-28">기수 / 학과</th>
              <th className="px-4 py-3 font-medium">졸업연도</th>
              <th className="px-4 py-3 font-medium">신청 유형</th>
              <th className="px-4 py-3 font-medium w-28">신청일</th>
              <th className="px-4 py-3 font-medium w-48 text-center">작업</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {isError ? (
              <ErrorState colSpan={7} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-cool-gray">
                  로딩 중...
                </td>
              </tr>
            ) : items.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-cool-gray">
                  가입 신청이 없습니다.
                </td>
              </tr>
            ) : (
              items.map((m) => (
                <tr key={m.userSeq} className="border-b border-border-light hover:bg-background">
                  <td className="px-4 py-3 font-mono text-dark-slate">{m.userSeq}</td>
                  <td className="px-4 py-3 text-dark-slate">{m.userName}</td>
                  <td className="px-4 py-3 text-cool-gray">
                    {m.cohort ?? '—'}
                    {m.department ? <span className="ml-1 text-xs text-cool-gray/70">/ {m.department}</span> : null}
                  </td>
                  <td className="px-4 py-3 text-cool-gray">{m.graduationYear ?? '—'}</td>
                  <td className="px-4 py-3 text-cool-gray">{m.status === 'reapproval_pending' ? '재승인' : '최초 신청'}</td>
                  <td className="px-4 py-3 text-cool-gray">{m.submittedAt?.slice(0, 10) ?? '—'}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-center gap-2">
                      <Button
                        onClick={() => setPendingAction({ member: m, type: 'approve' })}
                        disabled={isActioning}
                        className="px-4"
                      >
                        승인
                      </Button>
                      <Button
                        variant="destructive"
                        onClick={() => { setRejectionReason(''); setPendingAction({ member: m, type: 'reject' }); }}
                        disabled={isActioning}
                        className="px-4"
                      >
                        거절
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <ConfirmDialog
        open={pendingAction !== null}
        onOpenChange={(open) => { if (!open) setPendingAction(null); }}
        title={pendingAction?.type === 'approve' ? '회원 승인' : '가입 거절'}
        description={
          pendingAction?.type === 'approve'
            ? `${pendingAction.member.userName}님의 가입을 승인하시겠습니까?`
            : `${pendingAction?.member.userName}님의 가입 신청을 거절하시겠습니까?`
        }
        confirmLabel={pendingAction?.type === 'approve' ? '승인' : '거절'}
        variant={pendingAction?.type === 'reject' ? 'destructive' : 'default'}
        onConfirm={handleConfirm}
        isPending={isActioning}
        confirmDisabled={isRejectingAction && !rejectionReason.trim()}
      >
        {isRejectingAction && (
          <label className="mt-4 block text-sm text-dark-slate">
            거절 사유
            <textarea value={rejectionReason} onChange={(event) => setRejectionReason(event.target.value)}
              className="mt-2 w-full rounded-lg border border-border-light p-3" rows={3} />
          </label>
        )}
      </ConfirmDialog>
    </div>
  );
}
