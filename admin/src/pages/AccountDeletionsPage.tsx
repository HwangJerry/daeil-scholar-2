// AccountDeletionsPage — Deletion queue where operators review exact affected records before processing.
import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AccountErasurePreview } from '../components/AccountErasurePreview';
import { AccountErasureReceiptWork } from '../components/AccountErasureReceiptWork';
import { AccountErasureTargets } from '../components/AccountErasureTargets';
import { blockerMessage } from '../components/accountErasureLabels';
import { Button } from '../components/ui/Button';
import { fetchAccountDeletions, resolveAccountDeletion, type AccountDeletion, type DeletionStatus } from '../api/accountDeletions';

const QUEUE_PAGE_SIZE = 50;
const QUEUE_REFRESH_MS = 60_000;
const STATUS_LABELS: Record<DeletionStatus, string> = { pending: '접수', processing: '처리 중', completed: '완료', cancelled: '신청 취소' };
type ControlAction = 'automatic' | 'manual' | 'schedule' | 'retry_social' | 'cancel_verified';

function useCurrentTime() {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), QUEUE_REFRESH_MS);
    return () => clearInterval(timer);
  }, []);
  return now;
}

function ProcessingStatus({ item }: { item: AccountDeletion }) {
  const mode = item.processingMode === 'automatic' ? '자동 · 예약 시각 이후 자동 처리' : '관리자 검토 후 처리 · 자동 처리 중지됨';
  return (
    <>
      <p className="text-sm text-dark-slate">처리 방식: {mode} · 자동 작업 상태: {item.autoStage || '대기'}</p>
      {item.automationUpdatedAt && <p className="text-sm text-cool-gray">최근 상태 변경 {new Date(item.automationUpdatedAt).toLocaleString('ko-KR')}{item.processingMode === 'automatic' && item.nextAttemptAt && ` · 다음 시도 ${new Date(item.nextAttemptAt).toLocaleString('ko-KR')}`}</p>}
      {item.databaseErased && <p className="text-sm text-dark-slate">앱 운영 DB 처리 완료 · 파일 및 외부 처리 확인 후 최종 완료됩니다.</p>}
      {item.contextExpiresAt && <p role="status" className="text-sm text-dark-slate">외부 확인용 정보 만료: {new Date(item.contextExpiresAt).toLocaleString('ko-KR')}. 영수증 업무가 진행 중이어도 이 기한을 확인해주세요. 기한이 지나면 자동 재개에 필요한 정보가 없어 수동 검토가 필요할 수 있습니다.</p>}
      {item.socialUnlinkStalled && <p role="alert" className="text-sm font-semibold text-dark-slate">Apple·카카오 연결 해제가 15분 넘게 끝나지 않았거나 실패했습니다. 서버 로그의 연결 해제 오류를 확인한 뒤 "소셜 설정 확인 후 재시도"를 눌러 주세요.</p>}
      {item.autoCode && <p role="status" className="text-sm text-dark-slate">{blockerMessage(item.autoCode)} <span className="break-all">({item.autoCode})</span> 원인을 해결하면 자동으로 재시도합니다.</p>}
    </>
  );
}

function DeletionReview({ item }: { item: AccountDeletion }) {
  const client = useQueryClient();
  const now = useCurrentTime();
  const [verificationNote, setVerificationNote] = useState('');
  const mutation = useMutation({
    mutationFn: (action: ControlAction) => resolveAccountDeletion(item.requestId, action === 'cancel_verified' ? { action, evidenceReference: verificationNote } : { action }),
    onSuccess: () => client.invalidateQueries({ queryKey: ['account-deletions'] }),
  });
  const active = item.status === 'pending' || item.status === 'processing';
  const overdue = active && new Date(item.dueAt).getTime() < now;

  return (
    <section className="space-y-4 rounded-xl border border-border-light bg-surface p-5" aria-label={`삭제 요청 ${item.requestId}`}>
      <h2 className="text-lg font-semibold text-dark-slate">접수번호 {item.requestId} · {STATUS_LABELS[item.status]}</h2>
      <p className="text-sm text-cool-gray">회원 번호 {item.userSeq ?? '삭제됨'} · 접수 {new Date(item.requestedAt).toLocaleString('ko-KR')}</p>
      <p className="text-sm text-dark-slate">자동 처리 예정 {item.scheduledAt ? new Date(item.scheduledAt).toLocaleString('ko-KR') : '기존 요청 · 예약 검토 필요'} · 결과 안내 기한 {new Date(item.dueAt).toLocaleDateString('ko-KR')}{overdue && ' · 기한 경과 — 즉시 확인 필요'}</p>
      {active && (
        <div className="space-y-3">
          <ProcessingStatus item={item} />
          {mutation.error && <p role="alert" className="text-sm text-dark-slate">{mutation.error instanceof Error ? mutation.error.message : '요청을 처리하지 못했습니다.'}</p>}
          <div className="flex flex-wrap gap-3">
            {!item.scheduledAt && <Button variant="outline" disabled={mutation.isPending} onClick={() => { if (window.confirm('현재 설정된 대기 기간을 적용해 이 기존 요청의 자동 처리를 예약할까요?')) mutation.mutate('schedule'); }}>기존 요청 예약 적용</Button>}
            {(item.autoCode === 'PROVIDER_REVOCATION_PENDING' || item.socialUnlinkStalled) && <Button variant="outline" disabled={mutation.isPending} onClick={() => { if (window.confirm('소셜 연결 해제 설정과 자격 증명을 확인했나요? 실패한 작업을 재시도합니다.')) mutation.mutate('retry_social'); }}>소셜 설정 확인 후 재시도</Button>}
            <Button variant="outline" disabled={mutation.isPending || item.processingMode === 'manual'} onClick={() => mutation.mutate('manual')}>자동 처리 중지 · 관리자 검토 후 처리</Button>
            <Button variant="outline" disabled={mutation.isPending || item.processingMode === 'automatic'} onClick={() => mutation.mutate('automatic')}>검토 없이 자동 처리 재개</Button>
          </div>
          {!item.databaseErased && <AccountErasurePreview item={item} />}
        </div>
      )}
      {item.canCancel && <div className="space-y-2">
        <label className="block text-sm">신청 취소 본인 확인 근거 (개인정보 제외)<input className="w-full rounded-lg border border-border-light p-2" value={verificationNote} onChange={event => setVerificationNote(event.target.value)} maxLength={150}/></label>
        <Button variant="outline" disabled={!verificationNote.trim() || mutation.isPending} onClick={() => { if (window.confirm('문의자의 본인 확인을 완료했나요? 신청을 취소하고 일반 계정 이용을 복원합니다.')) mutation.mutate('cancel_verified'); }}>본인 확인 후 신청 취소</Button>
      </div>}
      <AccountErasureReceiptWork item={item} />
      <AccountErasureTargets targets={item.targets} requestId={active ? item.requestId : undefined} />
      {item.status === 'completed' && <p className="text-sm text-cool-gray">완료 {item.completedAt && new Date(item.completedAt).toLocaleString('ko-KR')} · 보존 안내: {item.retainedRecords}</p>}
    </section>
  );
}

export function AccountDeletionsPage() {
  const [status, setStatus] = useState<DeletionStatus>('pending');
  const [before, setBefore] = useState(0);
  const query = useQuery({ queryKey: ['account-deletions', status, before], queryFn: () => fetchAccountDeletions(status, before), refetchInterval: QUEUE_REFRESH_MS });
  const items = query.data?.items ?? [];
  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-semibold text-dark-slate">계정 삭제 요청</h1>
      <p className="text-sm text-cool-gray">담당: 황제철 · ghkdwp018@gmail.com. 예약 시각 이후 자동 처리합니다. 최고 관리자는 ‘처리 대상 기록 검토’에서 탈퇴 시 삭제·익명화되는 기록을 직접 확인한 뒤 즉시 처리할 수 있습니다. 검토 후 처리하려면 먼저 자동 처리를 중지해 두세요. 실제 처리가 시작되기 전에는 사용자가 신청을 취소할 수 있고, 본인 확인 후 관리자 취소도 가능합니다. 취소·완료 이력은 다시 실행할 수 없습니다.</p>
      <label className="block text-sm text-dark-slate">상태{' '}
        <select value={status} onChange={(event) => { setStatus(event.target.value as DeletionStatus); setBefore(0); }} className="rounded-lg border border-border-light bg-surface p-2">
          {Object.entries(STATUS_LABELS).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>
      {query.isPending && <p role="status">불러오는 중...</p>}
      {query.isError && <p role="alert">{query.error.message}</p>}
      {query.isSuccess && items.length === 0 && <p>해당 상태의 요청이 없습니다.</p>}
      {items.map((item) => <DeletionReview key={item.requestId} item={item} />)}
      <div className="flex gap-3">
        {before > 0 && <Button variant="outline" onClick={() => setBefore(0)}>처음으로</Button>}
        {items.length === QUEUE_PAGE_SIZE && <Button variant="outline" onClick={() => setBefore(items[items.length - 1].requestId)}>다음 페이지</Button>}
      </div>
    </div>
  );
}
