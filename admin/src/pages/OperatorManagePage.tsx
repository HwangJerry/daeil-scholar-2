// OperatorManagePage — root-only screen to grant, change and revoke root/operator admin roles
import { useAuth } from '../hooks/useAuth.ts';
import { useConfirmDialog } from '../hooks/useConfirmDialog.ts';
import { useOperators } from '../hooks/useOperators.ts';
import type { AdminRole } from '../api/operators.ts';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { OperatorAddPanel } from '../components/operator/OperatorAddPanel.tsx';
import { OperatorTable } from '../components/operator/OperatorTable.tsx';
import { ADMIN_ROLE_OPTIONS, getAdminRoleLabel } from '../components/operator/operatorRoles.ts';

type PendingAction =
  | { kind: 'set'; usrSeq: number; name: string; role: AdminRole }
  | { kind: 'revoke'; usrSeq: number; name: string };

function describeAction(action: PendingAction | null) {
  if (!action) return '';
  if (action.kind === 'revoke') return `${action.name}님의 관리자 권한을 해제합니다. 다음 요청부터 관리자 화면에 들어올 수 없습니다.`;
  return `${action.name}님을 '${getAdminRoleLabel(action.role)}'(으)로 지정합니다.`;
}

export function OperatorManagePage() {
  const user = useAuth((s) => s.user);
  const isRoot = user?.adminRole === 'root';
  const { operators, isLoading, isError, refetch, setRole, revoke } = useOperators(isRoot);
  const confirm = useConfirmDialog<PendingAction>();
  const isMutating = setRole.isPending || revoke.isPending;

  if (!isRoot) {
    return (
      <div className="space-y-4">
        <h2 className="text-xl font-bold text-dark-slate">운영자 관리</h2>
        <p className="rounded-2xl border border-border-light bg-white p-6 text-sm text-cool-gray">
          root 관리자만 이 화면을 사용할 수 있습니다.
        </p>
      </div>
    );
  }

  const handleConfirm = () => {
    const action = confirm.confirm();
    if (!action) return;
    if (action.kind === 'revoke') revoke.mutate(action.usrSeq);
    else setRole.mutate({ usrSeq: action.usrSeq, role: action.role });
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold text-dark-slate">운영자 관리</h2>
        <ul className="mt-2 space-y-0.5 text-xs text-cool-gray">
          {ADMIN_ROLE_OPTIONS.map((option) => <li key={option.value}><b>{option.label}</b>: {option.description}</li>)}
          <li>본인의 권한은 바꿀 수 없고, root 관리자는 최소 1명 남아야 합니다.</li>
        </ul>
      </div>

      <OperatorAddPanel
        existingSeqs={new Set(operators.map((operator) => operator.usrSeq))}
        disabled={isMutating}
        onGrant={(member, role) => confirm.open({ kind: 'set', usrSeq: member.usrSeq, name: member.usrName, role })}
      />

      {isError ? (
        <table className="w-full"><tbody><ErrorState colSpan={1} onRetry={() => void refetch()} /></tbody></table>
      ) : isLoading ? (
        <p className="py-8 text-center text-sm text-cool-gray">로딩 중...</p>
      ) : (
        <OperatorTable
          operators={operators}
          currentUserSeq={user.usrSeq}
          disabled={isMutating}
          onRoleChange={(operator, role) => confirm.open({ kind: 'set', usrSeq: operator.usrSeq, name: operator.usrName, role })}
          onRevoke={(operator) => confirm.open({ kind: 'revoke', usrSeq: operator.usrSeq, name: operator.usrName })}
        />
      )}

      <ConfirmDialog
        open={confirm.isOpen}
        onOpenChange={(open) => { if (!open) confirm.close(); }}
        title={confirm.pendingValue?.kind === 'revoke' ? '관리자 권한 해제' : '관리자 권한 지정'}
        description={describeAction(confirm.pendingValue)}
        confirmLabel={confirm.pendingValue?.kind === 'revoke' ? '해제' : '지정'}
        variant={confirm.pendingValue?.kind === 'revoke' ? 'destructive' : 'default'}
        onConfirm={handleConfirm}
        isPending={isMutating}
      />
    </div>
  );
}
