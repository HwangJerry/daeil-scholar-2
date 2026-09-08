import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { Button } from '../components/ui/Button';

const PAGE_SIZE = 50;
const DETAIL_VISIBLE_MS = 5 * 60 * 1000;
type Archive = { id: number; basis: string; retainUntil: string };
type Evidence = {
  donorName: string; donationDate: string; grossAmount: number;
  refundedAmount: number; netAmount: number; source: string; transactionNumber: string;
  basisDate: string; evidenceReference: string; dateMeaning?: string;
};
const PURPOSES = { receipt_inquiry: '기부 증빙 문의 확인', accounting_review: '회계 검토', official_request: '공식 자료 요청 대응' };
const BASIS_LABELS: Record<string, string> = { ledger_10y: '기부 장부 증빙', receipt_5y: '기부금영수증 증빙' };

function ArchiveDetail({ item }: { item: Archive }) {
  const [purpose, setPurpose] = useState('');
  const [detail, setDetail] = useState<Evidence | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    if (!detail) return;
    const timer = window.setTimeout(() => setDetail(null), DETAIL_VISIBLE_MS);
    const hide = () => { if (document.hidden) setDetail(null); };
    document.addEventListener('visibilitychange', hide);
    return () => { window.clearTimeout(timer); document.removeEventListener('visibilitychange', hide); };
  }, [detail]);
  async function read() {
    setPending(true); setError(''); setDetail(null);
    try {
      const result = await api.post<Evidence>(`/api/admin/donation-archives/${item.id}/read`, { purpose });
      if (!document.hidden) setDetail(result);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '증빙을 조회하지 못했습니다.');
    } finally { setPending(false); }
  }
  return <section className="space-y-3 rounded-xl border border-border-light bg-surface p-5">
    <h2 className="font-semibold">보관 번호 {item.id}</h2>
    <p>{BASIS_LABELS[item.basis] ?? item.basis} · 보관 종료 {item.retainUntil}</p>
    <label className="block">조회 목적
      <select aria-label={`보관 번호 ${item.id} 조회 목적`} value={purpose} onChange={(event) => setPurpose(event.target.value)} className="ml-3 rounded-lg border border-border-light bg-surface p-2">
        <option value="">선택해주세요</option>
        {Object.entries(PURPOSES).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
      </select>
    </label>
    <Button disabled={!purpose || pending} onClick={() => void read()}>{pending ? '조회 중…' : '증빙 열람'}</Button>
    {error && <p role="alert">{error}</p>}
    {detail && <div className="space-y-3" aria-label="기부 증빙 상세">
      <dl className="space-y-2 break-words">
        <div><dt>기부자명</dt><dd>{detail.donorName || '미기록'}</dd></div>
        <div><dt>기부일</dt><dd>{detail.donationDate || '미기록'}{detail.dateMeaning && ' (등록일이며 실제 납부일 확인 필요)'}</dd></div>
        <div><dt>총액 / 환불액 / 실수령액</dt><dd>{detail.grossAmount.toLocaleString()}원 / {detail.refundedAmount.toLocaleString()}원 / {detail.netAmount.toLocaleString()}원</dd></div>
        <div><dt>기부 경로</dt><dd>{detail.source}</dd></div>
        <div><dt>거래번호</dt><dd>{detail.transactionNumber || '미기록'}</dd></div>
        <div><dt>보관 기산일</dt><dd>{detail.basisDate}</dd></div>
        <div><dt>증빙 참조</dt><dd>{detail.evidenceReference}</dd></div>
      </dl>
      <Button variant="outline" onClick={() => setDetail(null)}>증빙 닫기</Button>
    </div>}
  </section>;
}

export function DonationArchivesPage() {
  const [before, setBefore] = useState(0);
  const query = useQuery({ queryKey: ['donation-archives', before], queryFn: () => api.get<{ items: Archive[] }>(`/api/admin/donation-archives?before=${before}`), retry: false });
  const items = query.data?.items ?? [];
  return <div className="space-y-5 text-dark-slate">
    <h1 className="text-2xl font-semibold">탈퇴 회원 기부 증빙</h1>
    <p className="text-sm text-cool-gray">최고 관리자 전용입니다. 증빙을 열면 열람자·목적·시각이 기록됩니다. 상세 내용은 5분 후 또는 다른 탭으로 이동하면 닫힙니다.</p>
    {query.isPending && <p role="status">불러오는 중…</p>}
    {query.isError && <p role="alert">{query.error.message}</p>}
    {query.isSuccess && items.length === 0 && <p>보관 중인 증빙이 없습니다.</p>}
    <div key={before} className="space-y-4">{items.map((item) => <ArchiveDetail key={item.id} item={item} />)}</div>
    <div className="flex gap-3">
      {before > 0 && <Button variant="outline" onClick={() => setBefore(0)}>처음으로</Button>}
      {items.length === PAGE_SIZE && <Button variant="outline" onClick={() => setBefore(items[items.length - 1].id)}>다음 페이지</Button>}
    </div>
  </div>;
}
