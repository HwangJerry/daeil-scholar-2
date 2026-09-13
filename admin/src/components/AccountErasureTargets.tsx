// AccountErasureTargets — Storage-specific evidence and next actions without personal data.
import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Button } from './ui/Button';
import { resolveAccountDeletion } from '../api/accountDeletions';
import type { ErasureTarget } from '../api/accountDeletions';

const TARGET_LABELS: Record<string, string> = {
  backups: '백업·복원 방지',
  historical_files: '과거 업로드 파일',
  external_data: '외부 서비스·로그',
  other_identifiers: '이메일·연락처 등 간접 참조',
};
const STATUS_LABELS: Record<ErasureTarget['status'], string> = {
  pending: '확인 대기', running: '처리 중', complete: '검증 완료',
  not_applicable: '해당 없음 확인', manual: '담당자 확인 필요', failed: '재시도 필요',
};

function nextAction(target: ErasureTarget): string {
  if (target.status === 'complete' || target.status === 'not_applicable') return '확인 근거를 보존하며 이 작업은 반복하지 않습니다.';
  if (target.status === 'manual') return '황제철 담당자가 저장소 설정과 남은 자료를 확인하고 기존 수동 처리 절차로 해결해야 합니다.';
  if (target.status === 'failed') return '서버 담당자가 연동 설정 또는 오류를 확인해야 합니다. 다음 자동 시도에서 다시 확인합니다.';
  return '실제 삭제 또는 해당 자료가 없다는 근거를 확인한 뒤 완료합니다. 만료 예정만으로 완료하지 않습니다.';
}

export function AccountErasureTargets({ targets, requestId }: { targets?: ErasureTarget[]; requestId?: number }) {
  if (!targets?.length) return <p className="text-sm text-cool-gray">저장소별 진행 상태가 아직 제공되지 않았습니다.</p>;
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-dark-slate">저장소별 삭제 진행</h3>
      <ul className="space-y-3">
        {targets.map(target => (
          <li key={target.target} className="space-y-1 rounded-lg border border-border-light p-3 text-sm">
            <p className="font-semibold text-dark-slate">{TARGET_LABELS[target.target] ?? '추가 저장소'} · {STATUS_LABELS[target.status] ?? '확인 필요'}</p>
            <p className="text-cool-gray">{nextAction(target)}</p>
            {target.lastAttemptAt && <p className="text-cool-gray">최근 시도 {new Date(target.lastAttemptAt).toLocaleString('ko-KR')} · 누적 {target.attempts}회</p>}
            {requestId && !['complete', 'not_applicable'].includes(target.status) && <TargetReview requestId={requestId} target={target} />}
            {target.evidenceReference && <p className="break-all text-cool-gray">확인 근거: {target.evidenceReference}</p>}
          </li>
        ))}
      </ul>
    </div>
  );
}

function TargetReview({ requestId, target }: { requestId: number; target: ErasureTarget }) {
 const [evidence, setEvidence] = useState('');
 const client = useQueryClient();
 const mutation = useMutation({
  mutationFn: (targetStatus: 'manual' | 'complete' | 'not_applicable') => resolveAccountDeletion(requestId, { action: 'target', target: target.target, targetStatus, evidenceReference: evidence.trim() }),
  onSuccess: () => { setEvidence(''); void client.invalidateQueries({ queryKey: ['account-deletions'] }); },
 });
 function review(status: 'manual' | 'complete' | 'not_applicable') {
  const message = status === 'manual' ? '담당자와 처리 경로를 확보했나요? 내부 삭제가 시작될 수 있지만 이 항목은 완료로 처리되지 않습니다.' : '실제 삭제 또는 해당 자료가 없음을 확인했나요? 확인 근거를 기록하며 되돌릴 수 없습니다.';
  if (window.confirm(message)) mutation.mutate(status);
 }
 return <div className="space-y-2">
  <label className="block">처리 인계 또는 검증 근거 (개인정보 제외)
   <input className="w-full rounded-lg border border-border-light p-2" value={evidence} onChange={event => setEvidence(event.target.value)} maxLength={150} />
  </label>
  <div className="flex flex-wrap gap-2">
   <Button variant="outline" disabled={!evidence.trim() || mutation.isPending} onClick={() => review('manual')}>수동 처리 인계 확인</Button>
   <Button variant="outline" disabled={!evidence.trim() || mutation.isPending} onClick={() => review('complete')}>실제 삭제 확인</Button>
   <Button variant="outline" disabled={!evidence.trim() || mutation.isPending} onClick={() => review('not_applicable')}>자료 없음 확인</Button>
  </div>
  {mutation.error && <p role="alert">{mutation.error instanceof Error ? mutation.error.message : '저장하지 못했습니다.'}</p>}
 </div>;
}
