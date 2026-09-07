// AccountDeletionsPage — Manual erasure work tracking with independent completion evidence.
import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AccountErasureReceiptWork } from '../components/AccountErasureReceiptWork';
import { AccountErasureTargets } from '../components/AccountErasureTargets';
import { Button } from '../components/ui/Button';
import { fetchAccountDeletions, resolveAccountDeletion, verifyAccountDeletion, type AccountDeletion, type DeletionEvidence, type DeletionStatus } from '../api/accountDeletions';

const QUEUE_PAGE_SIZE = 50;
const QUEUE_REFRESH_MS = 60_000;
const STATUS_LABELS: Record<DeletionStatus, string> = { pending: '접수', processing: '처리 중', completed: '완료' };
const AUTO_BLOCKERS: Record<string, string> = {
  FILE_STILL_REFERENCED: '다른 회원이나 게시글에서 사용하는 파일입니다. 소유권과 공유 참조를 확인해야 합니다.',
  FILE_OWNERSHIP_REVIEW_REQUIRED: '파일을 참조한 기록만 있고 소유권을 확인할 수 없어 삭제를 보류했습니다.',
  EXTERNAL_FILE_HANDOFF_REVIEW_REQUIRED: '기존 파일 큐의 외부 주소를 확인하고 외부 저장소 처리 증빙을 확보해야 합니다.',
  RECEIPT_CONTACT_REVIEW_REQUIRED: '진행 중인 영수증 연락 업무와 원본 보관 위치를 확인해야 합니다.',
  RECEIPT_CONTACT_WORK_PENDING: '영수증 업무·결과 전달·업무용 연락처 정리 확인을 기다리고 있습니다.',
  ERASURE_TARGETS_REVIEW_REQUIRED: '저장소별 작업 등록 상태를 확인해야 합니다.',
  ERASURE_CONTEXT_KEY_REQUIRED: '외부 삭제 작업용 별도 암호화 키 설정이 필요합니다.',
  ERASURE_CONTEXT_UNREADABLE: '외부 삭제 작업의 암호화 정보 확인이 필요합니다.',
  ERASURE_CONTEXT_EXPIRED_REVIEW_REQUIRED: '외부 삭제 작업용 정보의 보관 기한이 지났습니다. 수동 확인이 필요합니다.',

  DONATION_RETENTION_REVIEW_REQUIRED: '기부 자료의 보존 근거와 기간을 확인해야 합니다.',
  INVALID_DONATION_RETENTION_DECISION: '기부 자료의 보존 날짜 또는 근거를 수정해야 합니다.',
  DONATION_ARCHIVE_KEY_REQUIRED: '기부 자료 보관소의 암호화 설정이 필요합니다.',
  EXTERNAL_ERASURE_PROCESSOR_REQUIRED: '외부 서비스·백업 삭제 연동이 필요합니다.',
  EXTERNAL_ERASURE_PENDING: '외부 서비스·백업 삭제 완료를 기다리고 있습니다.',
  PROVIDER_REVOCATION_PENDING: 'Apple·카카오 연결 해제 완료를 기다리고 있습니다.',
  UNHANDLED_ACCOUNT_REFERENCE: '자동 처리 범위 밖에 남은 회원 관련 자료를 확인해야 합니다.',
  BILLING_REVOCATION_REVIEW_REQUIRED: '기존 정기결제 연결을 확인해야 합니다.',
  FILE_PATH_REVIEW_REQUIRED: '파일 경로 또는 소유 관계 확인이 필요합니다.',
  FILE_DELETE_RETRY_REQUIRED: '파일 삭제를 다시 시도합니다.',
};
const CHECKS = [
  ['resultNotified', '삭제 검증 후 이용자에게 처리 결과·보존 내역·이의제기 방법을 개별 안내하고 증빙을 남겼습니다.'],
  ['filesErased', '프로필·업로드 파일과 공유 사본을 실제로 삭제했습니다.'],
  ['backupsErased', '백업에서도 삭제되었고 복원 시 삭제 재적용 절차를 확인했습니다.'],
  ['externalDataErased', 'Sentry 등 외부 서비스의 해당 개인 데이터를 삭제했거나 수집된 자료가 없음을 확인했습니다.'],
  ['otherIdentifiersChecked', '이메일·전화번호·소셜 식별자와 간접 참조를 추가 점검했습니다.'],
] as const;

function DeletionReview({ item }: { item: AccountDeletion }) {
  const client = useQueryClient();
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), QUEUE_REFRESH_MS);
    return () => clearInterval(timer);
  }, []);
  const [evidence, setEvidence] = useState<DeletionEvidence>({
    action: 'complete', resultNotified: false, filesErased: false, backupsErased: false, externalDataErased: false,
    otherIdentifiersChecked: false, evidenceReference: '', retainedRecords: '', retentionUntil: '',
  });
  const mutation = useMutation({
    mutationFn: (action: 'start' | 'complete' | 'automatic' | 'manual') => resolveAccountDeletion(item.requestId, action === 'complete' ? evidence : { action }),
    onSuccess: () => client.invalidateQueries({ queryKey: ['account-deletions'] }),
  });
  const verification = useQuery({ queryKey: ['account-deletion-verification', item.requestId], queryFn: () => verifyAccountDeletion(item.requestId), enabled: false });
  const ready = CHECKS.every(([key]) => evidence[key]) && evidence.evidenceReference.trim() && evidence.retainedRecords.trim();
  const overdue = item.status !== 'completed' && new Date(item.dueAt).getTime() < now;

  return (
    <section className="space-y-4 rounded-xl border border-border-light bg-surface p-5" aria-label={`삭제 요청 ${item.requestId}`}>
      <h2 className="text-lg font-semibold text-dark-slate">접수번호 {item.requestId} · {STATUS_LABELS[item.status]}</h2>
      <p className="text-sm text-cool-gray">회원 번호 {item.userSeq ?? '삭제됨'} · 접수 {new Date(item.requestedAt).toLocaleString('ko-KR')}</p>
      <p className="text-sm text-dark-slate">처리 목표 {new Date(item.targetAt).toLocaleDateString('ko-KR')} · 결과 안내 기한 {new Date(item.dueAt).toLocaleDateString('ko-KR')}{overdue && ' · 기한 경과 — 즉시 확인 필요'}</p>
      {item.status !== 'completed' && (
        <div className="space-y-3">
          <p className="text-sm text-dark-slate">처리 방식: {item.processingMode === 'automatic' ? '자동' : '수동'} · 자동 작업 상태: {item.autoStage || '대기'}</p>
          {item.automationUpdatedAt && <p className="text-sm text-cool-gray">최근 상태 변경 {new Date(item.automationUpdatedAt).toLocaleString('ko-KR')}{item.processingMode === 'automatic' && item.nextAttemptAt && ` · 다음 시도 ${new Date(item.nextAttemptAt).toLocaleString('ko-KR')}`}</p>}
          {item.databaseErased && <p className="text-sm text-dark-slate">앱 운영 DB 삭제 완료 · 파일 및 외부 처리 확인 후 최종 완료됩니다.</p>}
          {item.contextExpiresAt && <p role="status" className="text-sm text-dark-slate">외부 확인용 정보 만료: {new Date(item.contextExpiresAt).toLocaleString('ko-KR')}. 영수증 업무가 진행 중이어도 이 기한을 확인해주세요. 기한이 지나면 자동 재개에 필요한 정보가 없어 수동 검토가 필요할 수 있습니다.</p>}
          {item.autoCode && <p role="status" className="text-sm text-dark-slate">{AUTO_BLOCKERS[item.autoCode] ?? '서버 관리자의 확인이 필요합니다.'} <span className="break-all">({item.autoCode})</span> 원인을 해결하면 자동으로 재시도합니다.</p>}
          <div className="flex flex-wrap gap-3">
            <Button variant="outline" disabled={mutation.isPending || item.processingMode === 'manual'} onClick={() => mutation.mutate('manual')}>자동 처리 중지 · 수동으로 전환</Button>
            <Button variant="outline" disabled={mutation.isPending} onClick={() => mutation.mutate('automatic')}>자동 처리 시작 · 재개</Button>
          </div>
        </div>
      )}
      <AccountErasureReceiptWork item={item} />
      <AccountErasureTargets targets={item.targets} />
      {item.status === 'pending' && item.processingMode !== 'automatic' && <Button disabled={mutation.isPending} onClick={() => mutation.mutate('start')}>삭제 작업 시작 · 소셜 권한 철회 요청</Button>}
      {item.status === 'processing' && item.processingMode !== 'automatic' && (
        <>
          <p className="text-sm text-cool-gray">계정·관련 데이터는 운영 절차에 따라 수동으로 삭제하세요. 이 화면의 확인 버튼은 데이터를 삭제하지 않습니다. 법정 보존 자료는 별도 보관소에 옮기고 근거·범위·만료일을 기록해야 합니다.</p>
          <Button variant="outline" disabled={verification.isFetching} onClick={() => { void verification.refetch(); }}>남은 데이터 조회</Button>
          {verification.isError && <p role="alert">{verification.error.message}</p>}
          {verification.data && (
            <div className="overflow-x-auto">
              <p className="text-sm text-dark-slate">회원 번호로 확인되는 잔여 기록 {verification.data.items.length}개 항목. 이 조회가 파일·이메일·간접 참조 검토를 대신하지 않습니다.</p>
              <ul className="mt-2 space-y-1 text-sm text-cool-gray">{verification.data.items.map((row) => <li key={`${row.table}.${row.column}`}>{row.table}.{row.column}: {row.count}건</li>)}</ul>
            </div>
          )}
          <div className="space-y-3">{CHECKS.map(([key, label]) => (
            <label key={key} className="flex items-start gap-3 text-sm text-dark-slate"><input type="checkbox" checked={evidence[key]} onChange={(event) => setEvidence({ ...evidence, [key]: event.target.checked })} />{label}</label>
          ))}</div>
          <label className="block text-sm text-dark-slate">삭제 작업 확인 자료 번호·위치 (개인정보 원문 입력 금지)
            <textarea maxLength={1000} value={evidence.evidenceReference} onChange={(event) => setEvidence({ ...evidence, evidenceReference: event.target.value })} className="mt-2 w-full rounded-lg border border-border-light bg-surface p-3" />
          </label>
          <label className="block text-sm text-dark-slate">이용자에게 안내할 법정 보존 항목·근거·이의제기 방법 (없으면 ‘없음’)
            <textarea maxLength={1000} value={evidence.retainedRecords} onChange={(event) => setEvidence({ ...evidence, retainedRecords: event.target.value })} className="mt-2 w-full rounded-lg border border-border-light bg-surface p-3" />
          </label>
          <label className="block text-sm text-dark-slate">법정 보존 종료일 (항목별 기한은 별도 보존대장에 기록)
            <input type="date" value={evidence.retentionUntil} onChange={(event) => setEvidence({ ...evidence, retentionUntil: event.target.value })} className="ml-3 rounded-lg border border-border-light bg-surface p-2" />
          </label>
          <Button disabled={!ready || mutation.isPending} onClick={() => mutation.mutate('complete')}>실제 삭제 검증 후 완료 결과 게시</Button>
        </>
      )}
      {item.status === 'completed' && <p className="text-sm text-cool-gray">완료 {item.completedAt && new Date(item.completedAt).toLocaleString('ko-KR')} · 보존 안내: {item.retainedRecords}</p>}
      {mutation.isError && <p role="alert" className="text-sm text-dark-slate">{mutation.error.message}</p>}
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
      <p className="text-sm text-cool-gray">담당: 황제철 · ghkdwp018@naver.com. 통상 3일 내 처리, 접수일부터 10일 내 결과 안내. 작업 시작과 완료는 root 권한이 필요합니다. 새 요청은 자동으로 처리합니다. 기존 수동 요청은 수동 방식을 유지하며 언제든 전환할 수 있습니다. 자동 완료 결과는 앱의 확인번호로 게시합니다. 자동 이메일 발송은 없습니다. 수동 처리 시 담당자가 개별 통지하고 증빙을 남깁니다. 진행 중인 작업 단계가 끝난 뒤 수동 전환이 반영될 수 있습니다.</p>
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
