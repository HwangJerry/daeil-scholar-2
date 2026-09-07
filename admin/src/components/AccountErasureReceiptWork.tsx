// AccountErasureReceiptWork — Verify limited-purpose contact retention in accounting originals.
import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { resolveAccountDeletion, type AccountDeletion, type ReceiptWorkResolution } from '../api/accountDeletions';
import { Button } from './ui/Button';

const LABELS = { unreviewed: '미확인', not_required: '진행 중인 업무 없음', active: '영수증 업무 진행 중', completed: '결과 전달·연락처 정리 완료' };

export function AccountErasureReceiptWork({ item }: { item: AccountDeletion }) {
  const client = useQueryClient();
  const [choice, setChoice] = useState<'not_required' | 'active'>('not_required');
  const [storage, setStorage] = useState('separate_excel');
  const [evidence, setEvidence] = useState('');
  const [secured, setSecured] = useState(false);
  const [delivered, setDelivered] = useState(false);
  const [erased, setErased] = useState(false);
  const status = item.receiptWork?.status;
  const closing = status === 'active';
  const mutation = useMutation({
    mutationFn: () => {
      const request: ReceiptWorkResolution = {
        action: 'receipt_work', receiptWorkStatus: closing ? 'completed' : choice,
        originalStorage: storage, evidenceReference: evidence,
        contactSecured: secured, contactErased: erased, resultNotified: delivered,
      };
      return resolveAccountDeletion(item.requestId, request);
    },
    onSuccess: () => { setEvidence(''); void client.invalidateQueries({ queryKey: ['account-deletions'] }); },
  });
  if (!status) return null;
  const editable = item.status !== 'completed' && (status === 'unreviewed' || closing);
  const confirmed = closing ? delivered && erased : choice === 'not_required' || secured;
  return (
    <div className="space-y-3 rounded-lg border border-border-light p-4 text-sm text-dark-slate">
      <h3 className="font-semibold">영수증 연락 업무 · {LABELS[status]}</h3>
      <p className="text-cool-gray">담당: 황제철. 진행 중인 영수증 업무와 결과 전달이 끝나면 해당 업무용 연락처를 정리합니다. 원본의 법정 보존 증빙은 유지합니다. 연락처 원문은 이 화면에 입력하지 마세요.</p>
      {item.receiptWork?.evidenceReference && <p className="break-all text-cool-gray">확인 근거: {item.receiptWork.evidenceReference}</p>}
      {editable && (
        <>
          {!closing && (
            <>
              <label className="block">진행 중인 발급·정정·재발급 업무
                <select value={choice} onChange={event => setChoice(event.target.value as typeof choice)} className="ml-2 rounded border border-border-light bg-surface p-2">
                  <option value="not_required">없음</option><option value="active">있음</option>
                </select>
              </label>
              {choice === 'active' && (
                <>
                  <label className="block">업무용 원본 위치
                    <select value={storage} onChange={event => setStorage(event.target.value)} className="ml-2 rounded border border-border-light bg-surface p-2">
                      <option value="separate_excel">별도 회계 엑셀</option><option value="happy_nanum">해피나눔</option><option value="both">두 곳 모두</option>
                    </select>
                  </label>
                  <label className="flex items-start gap-2"><input type="checkbox" checked={secured} onChange={event => setSecured(event.target.checked)} />구체적인 진행 업무가 있으며, 앱 연락처를 삭제해도 처리할 수 있도록 원본에 필요한 연락처와 보존 근거를 확인했습니다.</label>
                </>
              )}
            </>
          )}
          {closing && (
            <>
              <label className="flex items-start gap-2"><input type="checkbox" checked={delivered} onChange={event => setDelivered(event.target.checked)} />발급·정정·재발급 업무와 결과 전달을 모두 완료했습니다.</label>
              <label className="flex items-start gap-2"><input type="checkbox" checked={erased} onChange={event => setErased(event.target.checked)} />해피나눔·회계 엑셀 등 해당 업무용 원본과 사본에서 불필요해진 연락처를 실제로 정리했습니다.</label>
            </>
          )}
          <label className="block">업무·원본 확인 또는 완료·정리 증빙 번호 (개인정보 원문 제외)
            <input value={evidence} maxLength={200} onChange={event => setEvidence(event.target.value)} className="mt-2 w-full rounded border border-border-light bg-surface p-2" />
          </label>
          <Button disabled={!confirmed || !evidence.trim() || mutation.isPending} onClick={() => mutation.mutate()}>{closing ? '결과 전달·연락처 정리 확인' : '영수증 업무 확인 저장'}</Button>
        </>
      )}
      {mutation.isError && <p role="alert">{mutation.error.message}</p>}
    </div>
  );
}
