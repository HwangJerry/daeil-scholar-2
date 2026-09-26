// OperatorTable — current admin role holders with role change and revoke controls (own row locked)
import type { AdminOperator, AdminRole } from '../../api/operators.ts';
import { formatSeoulDateTime } from '../../lib/formatSeoulDateTime.ts';
import { Button } from '../ui/Button.tsx';
import { Select } from '../ui/Select.tsx';
import { ADMIN_ROLE_OPTIONS } from './operatorRoles.ts';

interface OperatorTableProps {
  operators: AdminOperator[];
  currentUserSeq: number;
  disabled: boolean;
  onRoleChange: (operator: AdminOperator, role: AdminRole) => void;
  onRevoke: (operator: AdminOperator) => void;
}

export function OperatorTable({ operators, currentUserSeq, disabled, onRoleChange, onRevoke }: OperatorTableProps) {
  return (
    <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border-light text-left text-cool-gray">
            <th className="px-4 py-3 font-medium">이름</th>
            <th className="px-4 py-3 font-medium">아이디</th>
            <th className="px-4 py-3 font-medium w-44">권한</th>
            <th className="px-4 py-3 font-medium">최근 변경</th>
            <th className="px-4 py-3 font-medium w-24 text-right">해제</th>
          </tr>
        </thead>
        <tbody>
          {operators.length === 0 ? (
            <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">관리자가 없습니다.</td></tr>
          ) : operators.map((operator) => {
            const isSelf = operator.usrSeq === currentUserSeq;
            return (
              <tr key={operator.usrSeq} className="border-b border-border-light">
                <td className="px-4 py-3 text-dark-slate">
                  {operator.usrName}
                  {isSelf && <span className="ml-2 text-xs text-cool-gray">(본인)</span>}
                </td>
                <td className="px-4 py-3 text-cool-gray">{operator.usrId}</td>
                <td className="px-4 py-3">
                  <Select
                    aria-label={`${operator.usrName} 권한`}
                    value={operator.adminRole}
                    disabled={disabled || isSelf}
                    onChange={(e) => onRoleChange(operator, e.target.value as AdminRole)}
                    className="h-9 py-1"
                  >
                    {ADMIN_ROLE_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                  </Select>
                </td>
                <td className="px-4 py-3 text-cool-gray">
                  {formatSeoulDateTime(operator.updatedAt)}
                  {operator.updatedByName && <span className="ml-1 text-xs">· {operator.updatedByName}</span>}
                </td>
                <td className="px-4 py-3 text-right">
                  <Button variant="ghost" size="sm" disabled={disabled || isSelf} onClick={() => onRevoke(operator)}>
                    해제
                  </Button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
