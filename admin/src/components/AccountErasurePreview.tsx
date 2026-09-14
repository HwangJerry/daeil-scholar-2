// AccountErasurePreview — Operator review of the exact records an erasure changes, then plan-bound approval.
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button } from './ui/Button';
import { ErasurePreviewTable } from './ErasurePreviewTable';
import { blockerMessage, socialUnlinkLabel, tableLabel } from './accountErasureLabels';
import { expediteReviewedErasure, fetchErasurePreview, type ErasurePreview } from '../api/accountErasurePreview';
import type { AccountDeletion } from '../api/accountDeletions';

const PLAN_CHANGED_CODE = 'ERASURE_PLAN_CHANGED';

function isPlanChanged(error: unknown): boolean {
  return typeof error === 'object' && error !== null && (error as { code?: string }).code === PLAN_CHANGED_CODE;
}

function PreviewHolds({ preview }: { preview: ErasurePreview }) {
  if (preview.blockers.length === 0) return <p className="text-sm text-dark-slate">처리를 멈추게 할 보류 항목이 없습니다.</p>;
  return (
    <div role="alert" className="space-y-1 rounded-lg border border-border-light p-3 text-sm text-dark-slate">
      <p className="font-semibold">처리 보류 예정 항목 {preview.blockers.length}개 · 승인해도 해당 단계에서 멈춥니다.</p>
      <ul className="list-disc space-y-1 pl-5">
        {preview.blockers.map((code) => <li key={code}>{blockerMessage(code)} <span className="break-all text-cool-gray">({code})</span></li>)}
      </ul>
      {preview.unhandled.length > 0 && (
        <ul className="list-disc pl-5 text-cool-gray">
          {preview.unhandled.map((row) => <li key={`${row.table}.${row.column}`}>처리 범위 밖 기록: {tableLabel(row.table)} {row.table}.{row.column} {row.count}건</li>)}
        </ul>
      )}
    </div>
  );
}

function PreviewRecords({ preview }: { preview: ErasurePreview }) {
  const total = preview.tables.reduce((sum, table) => sum + table.count, 0);
  return (
    <div className="space-y-3">
      <p className="text-sm text-dark-slate">
        조회 시각 {new Date(preview.generatedAt).toLocaleString('ko-KR')} · 테이블 {preview.tables.length}개 · 기록 {total}건 · 삭제할 파일 {preview.files.length}개
      </p>
      <PreviewHolds preview={preview} />
      {(preview.socialUnlinks?.length ?? 0) > 0 && (
        <div className="space-y-1 rounded-lg border border-border-light p-3 text-sm text-dark-slate">
          <p className="font-semibold">소셜 연결 해제 · {preview.socialUnlinks?.length}건</p>
          <ul className="list-disc space-y-1 pl-5">{preview.socialUnlinks?.map((unlink) => <li key={unlink.provider}>{socialUnlinkLabel(unlink.provider, unlink.status)}</li>)}</ul>
          <p className="text-xs text-cool-gray">DB 기록을 지우기 전에 Apple·카카오에 연결 해제를 요청합니다. 해제가 끝나야 다음 단계로 넘어갑니다.</p>
        </div>
      )}
      {preview.tables.length === 0 && <p className="text-sm text-cool-gray">운영 DB에서 처리할 기록이 없습니다.</p>}
      <div className="space-y-2">{preview.tables.map((table) => <ErasurePreviewTable key={table.table} table={table} />)}</div>
      {preview.files.length > 0 && (
        <details className="rounded-lg border border-border-light">
          <summary className="cursor-pointer p-3 text-sm font-semibold text-dark-slate">삭제할 업로드 파일 · {preview.files.length}개</summary>
          <ul className="space-y-1 p-3 font-mono text-xs text-dark-slate">{preview.files.map((file) => <li key={file} className="break-all">{file}</li>)}</ul>
        </details>
      )}
      <p className="text-xs text-cool-gray">DB 처리 뒤에는 파일 삭제, Apple·카카오 연결 해제, 외부 서비스·백업 확인이 이어집니다. 진행 상황은 아래 저장소별 삭제 진행에서 확인합니다.</p>
    </div>
  );
}

export function AccountErasurePreview({ item }: { item: AccountDeletion }) {
  const client = useQueryClient();
  const [reviewed, setReviewed] = useState(false);
  const [notice, setNotice] = useState('');
  const preview = useQuery({ queryKey: ['account-erasure-preview', item.requestId], queryFn: () => fetchErasurePreview(item.requestId), enabled: false });
  const approval = useMutation({
    mutationFn: (digest: string) => expediteReviewedErasure(item.requestId, digest),
    onSuccess: () => { setReviewed(false); void client.invalidateQueries({ queryKey: ['account-deletions'] }); },
    onError: (error) => {
      if (!isPlanChanged(error)) return;
      setReviewed(false);
      setNotice('검토한 뒤 처리 대상 기록이 바뀌었습니다. 새로 불러온 목록을 다시 확인해주세요.');
      void preview.refetch();
    },
  });
  const load = () => { setReviewed(false); setNotice(''); void preview.refetch(); };
  const data = preview.data;
  const approve = () => {
    if (!data) return;
    const total = data.tables.reduce((sum, table) => sum + table.count, 0);
    if (window.confirm(`접수번호 ${item.requestId}의 대기 기간을 생략하고, 검토한 기록 ${total}건을 지금 자동 처리할까요? 삭제·익명화한 기록은 되돌릴 수 없습니다.`)) approval.mutate(data.planDigest);
  };
  return (
    <div className="space-y-3 rounded-lg border border-border-light p-4">
      <h3 className="text-sm font-semibold text-dark-slate">처리 대상 기록 검토</h3>
      <p className="text-sm text-cool-gray">탈퇴 처리 시 바뀌는 운영 DB 기록과 처리 방식(삭제·익명화)을 확인합니다. 조회만 하며 기록을 바꾸지 않습니다. 승인하면 검토한 목록 그대로 즉시 자동 처리하고, 그사이 기록이 바뀌었으면 승인을 거부합니다.</p>
      <Button variant="outline" disabled={preview.isFetching} onClick={load}>{data ? '대상 기록 다시 불러오기' : '처리 대상 기록 불러오기'}</Button>
      {preview.isFetching && <p role="status" className="text-sm text-cool-gray">대상 기록을 불러오는 중...</p>}
      {preview.isError && <p role="alert" className="text-sm text-dark-slate">{preview.error.message}</p>}
      {notice && <p role="status" className="text-sm text-dark-slate">{notice}</p>}
      {data && <PreviewRecords preview={data} />}
      {data && !item.expeditedAt && (
        <div className="space-y-2">
          <label className="flex items-start gap-3 text-sm text-dark-slate">
            <input type="checkbox" checked={reviewed} onChange={(event) => setReviewed(event.target.checked)} />
            위 대상 기록과 처리 방식(삭제·익명화)을 직접 확인했으며 영향 범위에 문제가 없습니다.
          </label>
          <Button disabled={!reviewed || approval.isPending || preview.isFetching} onClick={approve}>검토 완료 · 지금 탈퇴 처리</Button>
          {approval.isError && !isPlanChanged(approval.error) && <p role="alert" className="text-sm text-dark-slate">{approval.error.message}</p>}
        </div>
      )}
      {item.expeditedAt && <p className="text-sm text-cool-gray">검토 후 즉시 처리 요청됨 {new Date(item.expeditedAt).toLocaleString('ko-KR')}</p>}
    </div>
  );
}
