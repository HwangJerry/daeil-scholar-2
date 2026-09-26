// MemberDetailPage — member profile with admin correction dialog and confirmed status changes
import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Pencil } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { MemberProfileForm } from '../components/member/MemberProfileForm.tsx';
import { MemberStatusControl } from '../components/member/MemberStatusControl.tsx';
import { getMemberStatusLabel } from '../components/member/memberStatus.ts';
import { useMemberDetail } from '../hooks/useMemberDetail.ts';
import { useMemberStatusUpdate } from '../hooks/useMemberStatusUpdate.ts';
import { useConfirmDialog } from '../hooks/useConfirmDialog.ts';

export function MemberDetailPage() {
  const { seq } = useParams<{ seq: string }>();
  const navigate = useNavigate();
  const { data: member, isLoading } = useMemberDetail(seq);
  const { updateStatus, isUpdating } = useMemberStatusUpdate(seq);
  const statusDialog = useConfirmDialog<string>();
  const [isEditing, setIsEditing] = useState(false);

  if (isLoading) {
    return <div className="py-8 text-center text-cool-gray">로딩 중...</div>;
  }

  if (!member) {
    return <div className="py-8 text-center text-cool-gray">회원을 찾을 수 없습니다.</div>;
  }

  const pendingLabel = statusDialog.pendingValue ? getMemberStatusLabel(statusDialog.pendingValue) : '';

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
        <div className="mb-4 flex justify-end">
          <Button variant="outline" size="sm" onClick={() => setIsEditing(true)}>
            <Pencil className="mr-1.5 h-4 w-4" />
            정보 수정
          </Button>
        </div>
        <div className="space-y-4">
          <InfoRow label="이름" value={member.usrName} />
          <InfoRow label="아이디" value={member.usrId} />
          <InfoRow label="기수" value={member.usrFn || '—'} />
          <InfoRow label="학과" value={member.usrDept || '—'} />
          <InfoRow label="연락처" value={member.usrPhone || '—'} />
          <InfoRow label="이메일" value={member.usrEmail || '—'} />
          <InfoRow label="닉네임" value={member.usrNick || '—'} />
          <InfoRow label="가입일" value={member.regDate?.slice(0, 10) || '—'} />
          <InfoRow label="방문 횟수" value={String(member.visitCnt)} />
          <InfoRow label="최근 접속" value={member.visitDate?.slice(0, 10) || '—'} />

          <MemberStatusControl
            status={member.usrStatus}
            disabled={isUpdating}
            onChange={(status) => statusDialog.open(status)}
          />
        </div>
      </div>

      <MemberProfileForm open={isEditing} member={member} onOpenChange={setIsEditing} />

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
