// MemberStatusControl — status dropdown offering only the account statuses the API accepts (탈퇴/휴면/정지)
import { Badge } from '../ui/Badge.tsx';
import { Select } from '../ui/Select.tsx';
import { EDITABLE_MEMBER_STATUSES, getMemberStatusLabel, isEditableMemberStatus } from './memberStatus.ts';

interface MemberStatusControlProps {
  status: string;
  disabled: boolean;
  onChange: (status: string) => void;
}

export function MemberStatusControl({ status, disabled, onChange }: MemberStatusControlProps) {
  const isCurrentEditable = isEditableMemberStatus(status);

  return (
    <div className="border-t border-border-light pt-4">
      <div className="flex items-center justify-between">
        <label htmlFor="member-status" className="text-sm font-medium text-cool-gray">상태 변경</label>
        <div className="flex items-center gap-2">
          <Badge variant="muted">{getMemberStatusLabel(status)}</Badge>
          <Select
            id="member-status"
            value={status}
            onChange={(e) => { if (e.target.value !== status) onChange(e.target.value); }}
            disabled={disabled}
            className="w-32"
          >
            {!isCurrentEditable && (
              <option value={status} disabled>{getMemberStatusLabel(status)} (현재)</option>
            )}
            {EDITABLE_MEMBER_STATUSES.map((value) => (
              <option key={value} value={value}>{getMemberStatusLabel(value)}</option>
            ))}
          </Select>
        </div>
      </div>
      <p className="mt-2 text-xs text-cool-gray">승인·반려는 가입 신청 화면에서 처리합니다.</p>
    </div>
  );
}
