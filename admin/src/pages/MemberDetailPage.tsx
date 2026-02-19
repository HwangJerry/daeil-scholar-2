// MemberDetailPage — displays member profile and allows status changes with confirmation dialog
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Select } from '../components/ui/Select.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { useMemberDetail } from '../hooks/useMemberDetail.ts';
import { useMemberStatusUpdate } from '../hooks/useMemberStatusUpdate.ts';
import { useConfirmDialog } from '../hooks/useConfirmDialog.ts';

const STATUS_OPTIONS = [
  { value: 'AAA', label: '탈퇴' },
  { value: 'ABA', label: '휴면' },
  { value: 'ACA', label: '정지' },
  { value: 'BAA', label: '승인거절' },
  { value: 'BBB', label: '승인대기' },
  { value: 'CCC', label: '승인회원' },
  { value: 'ZZZ', label: '운영자' },
];

export function MemberDetailPage() {
  const { seq } = useParams<{ seq: string }>();
  const navigate = useNavigate();
  const { data: member, isLoading } = useMemberDetail(seq);
  const { updateStatus, isUpdating } = useMemberStatusUpdate(seq);
  const statusDialog = useConfirmDialog<string>();

  if (isLoading) {
    return <div className="py-8 text-center text-cool-gray">로딩 중...</div>;
  }

  if (!member) {
    return <div className="py-8 text-center text-cool-gray">회원을 찾을 수 없습니다.</div>;
  }

  const pendingLabel = STATUS_OPTIONS.find((o) => o.value === statusDialog.pendingValue)?.label ?? '';

  const handleStatusChange = (newStatus: string) => {
    statusDialog.open(newStatus);
  };

  const handleStatusConfirm = () => {
    const status = statusDialog.confirm();
    if (status != null) updateStatus(status);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate('/member')}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h2 className="text-xl font-bold text-dark-slate">회원 상세</h2>
      </div>

      <div className="max-w-lg rounded-2xl border border-border-light bg-white p-6 shadow-sm">
        <div className="space-y-4">
          <InfoRow label="이름" value={member.usrName} />
          <InfoRow label="아이디" value={member.usrId} />
          <InfoRow label="기수" value={member.usrFn ?? '—'} />
          <InfoRow label="연락처" value={member.usrPhone ?? '—'} />
          <InfoRow label="이메일" value={member.usrEmail ?? '—'} />
          <InfoRow label="닉네임" value={member.usrNick ?? '—'} />
          <InfoRow label="가입일" value={member.regDate?.slice(0, 10) ?? '—'} />
          <InfoRow label="방문 횟수" value={String(member.visitCnt)} />
          <InfoRow label="최근 접속" value={member.visitDate?.slice(0, 10) ?? '—'} />

          <div className="flex items-center justify-between border-t border-border-light pt-4">
            <span className="text-sm font-medium text-cool-gray">상태 변경</span>
            <div className="flex items-center gap-2">
              <Badge variant="muted">{member.usrStatus}</Badge>
              <Select
                value={member.usrStatus}
                onChange={(e) => handleStatusChange(e.target.value)}
                disabled={isUpdating}
                className="w-28"
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </Select>
            </div>
          </div>
        </div>
      </div>

      <ConfirmDialog
        open={statusDialog.isOpen}
        onOpenChange={(open) => { if (!open) statusDialog.close(); }}
        title="상태 변경"
        description={`상태를 '${pendingLabel}'(으)로 변경하시겠습니까?`}
        confirmLabel="변경"
        variant="default"
        onConfirm={handleStatusConfirm}
        isPending={isUpdating}
      />
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-cool-gray">{label}</span>
      <span className="text-sm font-medium text-dark-slate">{value}</span>
    </div>
  );
}
