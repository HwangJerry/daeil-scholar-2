// MessageReportsPage — Restricted evidence review and recorded moderation decisions.
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchMessageReports, resolveMessageReport, type MessageReport, type ReportStatus } from '../api/messageReports';
import { Button } from '../components/ui/Button';

const REASON_LABELS: Record<string, string> = {
  harassment: '괴롭힘·욕설·위협', spam: '스팸·광고·사기', inappropriate: '부적절한 콘텐츠', other: '기타',
};
const STATUS_LABELS: Record<ReportStatus, string> = { open: '미처리', removed: '콘텐츠 삭제', dismissed: '위반 아님' };
const PAGE_SIZE = 50;

function ReportReview({ report }: { report: MessageReport }) {
  const [note, setNote] = useState('');
  const client = useQueryClient();
  const resolve = useMutation({
    mutationFn: (status: 'removed' | 'dismissed') => resolveMessageReport(report.id, status, note.trim()),
    onSuccess: () => client.invalidateQueries({ queryKey: ['message-reports'] }),
  });

  return (
    <article className="rounded-xl border border-border-light bg-surface p-5">
      <div className="flex flex-wrap justify-between gap-3">
        <h2 className="font-semibold text-dark-slate">신고 #{report.id} · {REASON_LABELS[report.reason] ?? report.reason}</h2>
        <span className="text-sm text-cool-gray">{new Date(report.createdAt).toLocaleString('ko-KR')}</span>
      </div>
      <p className="mt-3 text-sm text-cool-gray">
        신고자 #{report.reporterSeq} · 대상 회원{' '}
        <Link to={`/member/${report.reportedSeq}`} className="underline">#{report.reportedSeq} 관리</Link>
      </p>
      <blockquote className="mt-4 whitespace-pre-wrap break-words rounded-lg bg-background p-4 text-dark-slate">{report.content}</blockquote>
      <p className="mt-3 whitespace-pre-wrap break-words text-sm text-dark-slate">추가 설명: {report.details || '없음'}</p>
      {report.status === 'open' ? (
        <div className="mt-4 space-y-3">
          <label className="block text-sm text-dark-slate">
            처리 사유 (필수)
            <textarea value={note} onChange={(event) => setNote(event.target.value)} maxLength={1000}
              className="mt-2 block min-h-24 w-full rounded-lg border border-border-light bg-surface p-3" disabled={resolve.isPending} />
          </label>
          <p className="text-sm text-cool-gray">콘텐츠 삭제를 선택하면 대화에서 원문을 제거합니다. 신고 원문은 이 검토 기록에 남습니다.</p>
          <div className="flex flex-wrap gap-3">
            <Button disabled={!note.trim() || resolve.isPending} onClick={() => resolve.mutate('removed')}>위반 콘텐츠 삭제</Button>
            <Button variant="outline" disabled={!note.trim() || resolve.isPending} onClick={() => resolve.mutate('dismissed')}>위반 아님으로 종결</Button>
          </div>
          {resolve.isError && <p role="alert" className="text-sm text-error">{resolve.error.message}</p>}
        </div>
      ) : <p className="mt-4 text-sm text-dark-slate">{STATUS_LABELS[report.status]}: {report.moderatorNote}</p>}
    </article>
  );
}

export function MessageReportsPage() {
  const [status, setStatus] = useState<ReportStatus>('open');
  const [before, setBefore] = useState(0);
  const reports = useQuery({
    queryKey: ['message-reports', status, before],
    queryFn: () => fetchMessageReports(status, before),
    refetchInterval: 60_000,
  });
  const items = reports.data?.items ?? [];

  return (
    <main className="space-y-5">
      <h1 className="text-2xl font-semibold text-dark-slate">메시지 신고 관리</h1>
      <p className="text-sm text-cool-gray">미처리 신고를 정기적으로 확인하고 처리 사유를 기록하세요. 반복 위반은 대상 회원 관리에서 조치할 수 있습니다.</p>
      <div className="flex flex-wrap gap-3">
        <label className="text-sm text-dark-slate">처리 상태{' '}
          <select value={status} onChange={(event) => { setStatus(event.target.value as ReportStatus); setBefore(0); }}
            className="rounded-lg border border-border-light bg-surface p-2">
            {Object.entries(STATUS_LABELS).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
        <Button variant="outline" onClick={() => { void reports.refetch(); }}>새로고침</Button>
      </div>
      {reports.isPending && <p role="status">신고를 불러오는 중...</p>}
      {reports.isError && <p role="alert">신고 목록을 불러오지 못했습니다. {reports.error.message}</p>}
      {reports.isSuccess && items.length === 0 && <p>해당 상태의 신고가 없습니다.</p>}
      {items.map((report) => <ReportReview key={report.id} report={report} />)}
      <div className="flex gap-3">
        {before > 0 && <Button variant="outline" onClick={() => setBefore(0)}>처음으로</Button>}
        {items.length === PAGE_SIZE && <Button variant="outline" onClick={() => setBefore(items[items.length - 1].id)}>다음 페이지</Button>}
      </div>
    </main>
  );
}
