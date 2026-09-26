// memberStatus — member status labels and the statuses the admin status API accepts
const STATUS_LABELS: Record<string, string> = {
  AAA: '탈퇴',
  ABA: '휴면',
  ACA: '정지',
  BAA: '승인거절',
  BBB: '승인대기',
  CCC: '승인회원',
  ZZZ: '운영자',
};

// PUT /api/admin/member/{seq} rejects verification statuses (BAA/BBB/CCC/ZZZ) with 409;
// approval and rejection belong to the 가입 신청 screen.
export const EDITABLE_MEMBER_STATUSES = ['AAA', 'ABA', 'ACA'] as const;

export function getMemberStatusLabel(status: string) {
  return STATUS_LABELS[status] ?? status;
}

export function isEditableMemberStatus(status: string) {
  return (EDITABLE_MEMBER_STATUSES as readonly string[]).includes(status);
}
