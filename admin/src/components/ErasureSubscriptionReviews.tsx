import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Button } from './ui/Button';
import { reviewSubscriptionClosure } from '../api/accountErasurePreview';
import type { ErasureSubscriptionReview } from '../api/accountErasurePreview';

const REVIEW_EVIDENCE_MAX_CHARACTERS = 150;

export function ErasureSubscriptionReviews({ requestId, items, onReviewed }: { requestId: number; items: ErasureSubscriptionReview[]; onReviewed: () => void }) {
  if (!items.length) return null;
  return <div className="space-y-2 rounded-lg border border-border-light p-3 text-sm text-dark-slate">
    <p className="font-semibold">구독 참조 종료 검토</p>
    <p>외부 결제·청구가 종료되었음을 공급자 관리 화면 또는 확인 기록에서 검증한 뒤 근거를 남겨 주세요. 조회 실패·미확정 결제·청구키가 남아 있으면 계속 보류합니다.</p>
    {items.map(item => <SubscriptionReview key={`${item.subscriptionId}.${item.sourceFingerprint}`} item={item} requestId={requestId} onReviewed={onReviewed} />)}
  </div>;
}
function SubscriptionReview({ item, requestId, onReviewed }: { item: ErasureSubscriptionReview; requestId: number; onReviewed: () => void }) {
  const [evidence, setEvidence] = useState('');
  const [confirmed, setConfirmed] = useState(false);
  const [outcome, setOutcome] = useState<'' | 'failed' | 'cancelled'>('');
  const review = useMutation({
    mutationFn: () => {
      if (!outcome) throw new Error('공급자의 종료 상태를 확인해 주세요.');
      return reviewSubscriptionClosure(requestId, item.subscriptionId, item.sourceFingerprint, evidence.trim(), outcome);
    },
    onSuccess: onReviewed,
  });
  return <div className="space-y-2 border-t border-border-light pt-2">
    <p>구독 {item.subscriptionId} · {item.status} · {item.hasBillingKey ? '청구키 잔류' : '청구키 없음'} · {item.reviewed ? '종료 확인 기록됨' : '종료 확인 필요'}</p>
    {item.canReview && !item.reviewed && <>
      <label className="block">구독 {item.subscriptionId} 종료 확인 근거
        <input className="w-full rounded-lg border border-border-light p-2" maxLength={REVIEW_EVIDENCE_MAX_CHARACTERS} value={evidence} onChange={event => setEvidence(event.target.value)} placeholder="개인정보·청구키 없이 확인 기록 번호 입력" />
      </label>
      <label className="block">구독 {item.subscriptionId} 공급자 종료 상태
        <select className="w-full rounded-lg border border-border-light p-2" value={outcome} onChange={event => setOutcome(event.target.value as '' | 'failed' | 'cancelled')}>
          <option value="">확인된 종료 상태 선택</option><option value="failed">최종 실패 · 추가 결제 가능성 없음</option><option value="cancelled">취소 완료 · 추가 결제 가능성 없음</option>
        </select>
      </label>
      <label className="flex gap-2"><input type="checkbox" checked={confirmed} onChange={event => setConfirmed(event.target.checked)} />구독 {item.subscriptionId} 외부 결제·청구 종료 확인</label>
      <Button variant="outline" disabled={!confirmed || !outcome || !evidence.trim() || review.isPending} onClick={() => review.mutate()}>구독 {item.subscriptionId} 종료 확인 기록</Button>
      {review.error && <p role="alert">{review.error.message} 대상 기록을 다시 불러와 확인해 주세요.</p>}
    </>}
  </div>;
}
