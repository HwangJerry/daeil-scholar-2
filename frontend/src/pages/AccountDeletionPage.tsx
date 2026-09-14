// AccountDeletionPage — Private receipt lookup without restoring the deleted account's session.
import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api/client';
import { CancelDeletionButton } from '../components/accountDeletion/CancelDeletionButton';
import { AccountDeletionRequestGuide } from '../components/accountDeletion/AccountDeletionRequestGuide';
import { PageMeta } from '../components/seo/PageMeta';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { PRIVACY_CONTACT } from '../domains/privacy/policyContent';

type Receipt = {
 scheduledAt?: string | null;
 expeditedAt?: string | null;
 needsAttention?: boolean;
  requestId: number;
  status: 'pending' | 'processing' | 'completed' | 'cancelled';
  canCancel?: boolean;
  cancelledAt?: string | null;
  databaseErased?: boolean;
  receiptWorkPending?: boolean;
  requestedAt: string;
  targetAt: string;
  dueAt: string;
  completedAt: string | null;
  retainedRecords: string;
  retentionUntil: string | null;
};
const STATUS_LABELS = { pending: '삭제 요청 접수', processing: '계정 삭제 처리 중입니다', completed: '계정 삭제 완료', cancelled: '탈퇴 신청이 취소되었습니다' };
const dateLabel = (value: string) => new Date(value).toLocaleDateString('ko-KR');

export function AccountDeletionPage() {
  const [token, setToken] = useState(() => new URLSearchParams(window.location.hash.slice(1)).get('receipt') ?? '');
  const [cancelToken, setCancelToken] = useState(() => new URLSearchParams(window.location.hash.slice(1)).get('cancel') ?? '');
  const [cancelling, setCancelling] = useState(false);
  const [receipt, setReceipt] = useState<Receipt | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const didLoad = useRef(false);

  async function lookup(value: string) {
    setError('');
    setReceipt(null);
    if (!/^[a-f\d]{64}$/i.test(value)) { setError('앱이나 담당자 회신으로 받은 확인번호 64자리를 입력해주세요.'); return; }
    setLoading(true);
    try {
      setReceipt(await api.post<Receipt>('/api/account-deletion/receipt', { receiptToken: value }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '처리 현황을 불러오지 못했습니다. 다시 시도해주세요.');
    } finally { setLoading(false); }
  }

  useEffect(() => {
    // The fragment is never sent to the server; remove it from browser history
    // immediately. Keep the secret only in component memory, not localStorage.
    if (didLoad.current) return;
    didLoad.current = true;
    window.history.replaceState(window.history.state, '', window.location.pathname + window.location.search);
    if (token) void lookup(token);
  }, [token]);

  async function cancelRequest() {
    if (cancelling || !receipt?.canCancel) return;
    setCancelling(true); setError('');
    try {
      const result = await api.post<Receipt>('/api/account-deletion/cancel', { receiptToken: token.trim(), cancelToken: cancelToken.trim() });
      setReceipt(result); setCancelToken('');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '취소 결과를 확인하지 못했습니다. 처리 현황을 확인한 뒤 다시 시도해주세요.');
      try { setReceipt(await api.post<Receipt>('/api/account-deletion/receipt', { receiptToken: token.trim() })); } catch { /* Keep the prior receipt, never claim cancellation. */ }
    } finally { setCancelling(false); }
  }
  function submit(event: FormEvent) { event.preventDefault(); void lookup(token.trim()); }

  return (
    <>
      <PageMeta title="계정 삭제 요청과 처리 현황" canonicalPath="/account-deletion" noIndex />
      <div className="mx-auto max-w-2xl space-y-8 px-5 py-14 sm:px-8 md:py-20">
        <header>
          <p className="text-xs font-semibold tracking-widest text-text-secondary">DFLH ACCOUNT</p>
          <h1 className="mt-4 font-serif text-3xl font-semibold text-text-primary sm:text-4xl">계정 삭제 요청과 처리 현황</h1>
          <p className="mt-5 leading-8 text-text-secondary">앱 없이 계정 삭제를 요청하는 방법과, 받은 확인번호로 처리 결과를 확인하는 방법을 안내합니다. 로그인은 필요하지 않습니다.</p>
        </header>
        <AccountDeletionRequestGuide />
        <h2 className="text-xl font-semibold text-text-primary">처리 현황 확인</h2>
        <form onSubmit={submit} className="space-y-4">
          <label htmlFor="deletion-receipt" className="block text-sm font-semibold text-text-primary">삭제 요청 확인번호</label>
          <input id="deletion-receipt" type="password" autoComplete="off" spellCheck={false} value={token} onChange={(event) => setToken(event.target.value)} maxLength={64} className="min-h-12 w-full rounded-md border border-border bg-surface px-4 text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary" aria-describedby="receipt-help" />
          <p id="receipt-help" className="text-sm leading-7 text-text-secondary">확인번호를 가진 사람은 결과를 볼 수 있으니 다른 사람에게 공유하지 마세요. 결과는 처리 완료 후 30일간 조회할 수 있습니다.</p>
          <Button type="submit" disabled={loading}>{loading ? '확인 중…' : '처리 현황 확인'}</Button>
        </form>
        {error && <p role="alert" className="text-sm text-text-primary">{error}</p>}
        {receipt && (
          <Card className="space-y-4 border-border p-6 shadow-none" role="status">
            <h2 className="text-xl font-semibold text-primary">{receipt.status === 'cancelled' ? STATUS_LABELS.cancelled : receipt.status === 'completed' ? STATUS_LABELS.completed : receipt.needsAttention ? '추가 확인이 필요합니다' : receipt.status === 'pending' && receipt.scheduledAt && !receipt.expeditedAt ? '자동 탈퇴 예약 대기' : STATUS_LABELS[receipt.status]}</h2>
            {receipt.canCancel && <div className="space-y-3">
              <p>실제 탈퇴 처리가 시작되기 전까지 신청을 취소할 수 있습니다.</p>
              <label className="block">취소 인증번호<input type="password" autoComplete="off" maxLength={64} value={cancelToken} onChange={event => setCancelToken(event.target.value)} className="w-full rounded-md border border-border bg-surface p-3" /></label>
              <CancelDeletionButton disabled={!/^[a-f\d]{64}$/i.test(cancelToken.trim())} cancelling={cancelling} onConfirm={cancelRequest} />
              {!cancelToken && <p>신청한 앱에서 취소하거나 별도로 보관한 취소 인증번호를 입력하세요. 인증번호를 분실했다면 문의 창구에서 본인 확인을 받아주세요.</p>}
            </div>}
            {receipt.receiptWorkPending && <p className="text-sm leading-7 text-text-secondary">진행 중인 영수증 업무를 처리하고 있습니다. 업무와 결과 전달이 끝나면 해당 업무용 연락처를 정리합니다. 다른 삭제 작업은 예약 및 검토 상태에 따라 진행합니다.</p>}
            <p className="text-sm text-text-secondary">접수번호 {receipt.requestId} · 접수일 {dateLabel(receipt.requestedAt)}</p>
            {receipt.status === 'cancelled' ? <p>신청이 취소되었습니다. 다시 로그인해 주세요. 관리자 권한은 별도 확인이 필요합니다. <Link to="/login">로그인으로 이동</Link></p> : receipt.status === 'completed' ? (
              <>
                <p className="leading-7 text-text-secondary">{receipt.completedAt && dateLabel(receipt.completedAt)}에 계정과 삭제 대상 정보의 삭제를 확인했습니다.</p>
                <p className="whitespace-pre-wrap break-words text-sm leading-7 text-text-secondary">법정 보존 안내: {receipt.retainedRecords}</p>
                {receipt.retentionUntil && <p className="text-sm text-text-secondary">보존 종료일: {dateLabel(receipt.retentionUntil)}. 항목별 기한은 위 안내를 확인해주세요.</p>}
              </>
            ) : (
              <p className="leading-8 text-text-secondary">{receipt.databaseErased ? '앱 운영 데이터는 삭제했습니다. 파일·외부 저장소의 남은 처리를 확인 중이며 최종 완료 전입니다.' : '계정 이용은 중지되었습니다. 예약 시각 이후 자동 처리하며 관리자가 먼저 처리를 시작할 수 있습니다.'} {receipt.scheduledAt ? `자동 처리 예정: ${new Date(receipt.scheduledAt).toLocaleString('ko-KR', { timeZone: 'Asia/Seoul' })} (한국 시간).` : '자동 처리 일정은 담당자가 확인 중입니다.'} {receipt.expeditedAt && '관리자가 조기 처리를 요청했습니다.'} {dateLabel(receipt.dueAt)}까지 결과를 안내합니다. 법령에 따라 보존할 자료는 범위와 근거를 별도로 안내합니다.</p>
            )}
          </Card>
        )}
        <div className="space-y-3 border-t border-border pt-6 text-sm leading-7 text-text-secondary">
          <p>처리 결과에 관한 문의·이의제기: {PRIVACY_CONTACT.requestHandler} · <a className="inline-block break-all text-primary underline underline-offset-4" href={`mailto:${PRIVACY_CONTACT.email}`}>{PRIVACY_CONTACT.email}</a></p>
          <Link className="inline-flex min-h-11 items-center text-primary underline underline-offset-4" to="/privacy">개인정보처리방침 보기</Link>
        </div>
      </div>
    </>
  );
}
