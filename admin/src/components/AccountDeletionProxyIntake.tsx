// AccountDeletionProxyIntake — Register an emailed deletion request after the coordinator verifies identity.
import { useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Button } from './ui/Button';
import { createAccountDeletionOnBehalf, type ProxyIntakeResult } from '../api/accountDeletions';

const EVIDENCE_MAX = 150;
const STATUS_PAGE_URL = 'https://daeilfoundation.or.kr/account-deletion';

function IssuedTokens({ result }: { result: ProxyIntakeResult }) {
  return (
    <div role="status" className="space-y-2 rounded-lg border border-border-light p-4 text-sm text-dark-slate">
      <p className="font-semibold">접수번호 {result.receipt.requestId} 접수 완료</p>
      <p>아래 두 번호를 회원에게 회신 메일로 보내 주세요. 이 화면을 벗어나면 다시 볼 수 없습니다. 확인번호로 처리 현황을 조회하고, 취소 인증번호로 처리가 시작되기 전까지 신청을 취소할 수 있습니다.</p>
      <label htmlFor="proxy-receipt-token" className="block">처리 현황 확인번호
        <input id="proxy-receipt-token" readOnly value={result.receiptToken} onFocus={(event) => event.currentTarget.select()} className="mt-1 w-full rounded border border-border-light p-2 font-mono text-xs" />
      </label>
      <label htmlFor="proxy-cancel-token" className="block">취소 인증번호
        <input id="proxy-cancel-token" readOnly value={result.cancelToken} onFocus={(event) => event.currentTarget.select()} className="mt-1 w-full rounded border border-border-light p-2 font-mono text-xs" />
      </label>
      <p className="text-cool-gray">처리 현황 페이지: {STATUS_PAGE_URL}</p>
    </div>
  );
}

export function AccountDeletionProxyIntake() {
  const client = useQueryClient();
  const [userSeq, setUserSeq] = useState('');
  const [evidence, setEvidence] = useState('');
  const [result, setResult] = useState<ProxyIntakeResult | null>(null);
  const mutation = useMutation({
    mutationFn: () => createAccountDeletionOnBehalf(Number(userSeq), evidence.trim()),
    onSuccess: (data) => {
      setResult(data);
      setUserSeq('');
      setEvidence('');
      void client.invalidateQueries({ queryKey: ['account-deletions'] });
    },
  });
  const valid = /^[1-9]\d*$/.test(userSeq) && evidence.trim().length > 0;
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (!valid) return;
    if (window.confirm(`회원 번호 ${userSeq}의 계정 이용을 즉시 중지하고 탈퇴 요청을 대신 접수할까요? 본인 확인을 마친 요청만 접수하세요.`)) {
      mutation.mutate();
    }
  };
  return (
    <section aria-labelledby="proxy-intake" className="space-y-4 rounded-xl border border-border-light bg-surface p-5">
      <h2 id="proxy-intake" className="text-lg font-semibold text-dark-slate">이메일 요청 대신 접수</h2>
      <p className="text-sm text-cool-gray">앱 없이 이메일로 들어온 탈퇴 요청을 본인 확인 후 접수합니다. 회원 번호는 회원 관리의 회원 상세 화면 주소에서 확인할 수 있습니다. 접수하면 앱에서 요청한 것과 같이 계정 이용이 즉시 중지되고 예약 시각 이후 자동 처리됩니다.</p>
      <form onSubmit={submit} className="space-y-3">
        <label htmlFor="proxy-user-seq" className="block text-sm text-dark-slate">회원 번호
          <input id="proxy-user-seq" inputMode="numeric" value={userSeq} onChange={(event) => setUserSeq(event.target.value.trim())} className="mt-1 w-full rounded-lg border border-border-light p-2" />
        </label>
        <label htmlFor="proxy-evidence" className="block text-sm text-dark-slate">본인 확인 근거 (개인정보 제외. 예: 가입 이메일로 받은 요청 확인, 수신 일시)
          <input id="proxy-evidence" value={evidence} maxLength={EVIDENCE_MAX} onChange={(event) => setEvidence(event.target.value)} className="mt-1 w-full rounded-lg border border-border-light p-2" />
        </label>
        <Button type="submit" disabled={!valid || mutation.isPending}>{mutation.isPending ? '접수 중…' : '본인 확인 후 대신 접수'}</Button>
      </form>
      {mutation.isError && <p role="alert" className="text-sm text-dark-slate">{mutation.error instanceof Error ? mutation.error.message : '접수하지 못했습니다.'}</p>}
      {result && <IssuedTokens result={result} />}
    </section>
  );
}
