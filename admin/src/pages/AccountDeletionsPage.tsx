// AccountDeletionsPage — Manual erasure work tracking with independent completion evidence.
import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button } from '../components/ui/Button';
import { fetchAccountDeletions, resolveAccountDeletion, verifyAccountDeletion, type AccountDeletion, type DeletionEvidence, type DeletionStatus } from '../api/accountDeletions';

const QUEUE_PAGE_SIZE = 50;
const QUEUE_REFRESH_MS = 60_000;
const STATUS_LABELS: Record<DeletionStatus, string> = { pending: '접수', processing: '처리 중', completed: '완료' };
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
    mutationFn: (action: 'start' | 'complete') => resolveAccountDeletion(item.requestId, action === 'start' ? { action } : evidence),
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
      {item.status === 'pending' && <Button disabled={mutation.isPending} onClick={() => mutation.mutate('start')}>삭제 작업 시작 · 소셜 권한 철회 요청</Button>}
      {item.status === 'processing' && (
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
      <p className="text-sm text-cool-gray">담당: 황제철 · ghkdwp018@naver.com. 통상 3일 내 처리, 접수일부터 10일 내 결과 안내. 작업 시작과 완료는 root 권한이 필요합니다. 자동 이메일 발송은 없습니다. 담당자가 결과를 개별 통지하고 발송 증빙을 남겨야 합니다. 이용자는 앱의 확인번호로도 결과를 조회합니다.</p>
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
