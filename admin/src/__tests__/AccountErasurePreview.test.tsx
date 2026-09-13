// AccountErasurePreview.test — Operators see affected records and approve only the reviewed plan.
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api, ApiClientError } from '../api/client';
import { AccountDeletionsPage } from '../pages/AccountDeletionsPage';

afterEach(() => { cleanup(); vi.restoreAllMocks(); });

const DIGEST = 'a'.repeat(64);
const QUEUE_ITEM = { requestId: 7, userSeq: 42, status: 'pending', requestedAt: '2026-09-13T00:00:00Z', scheduledAt: '2026-09-16T00:00:00Z', targetAt: '2026-09-16T00:00:00Z', dueAt: '2026-09-26T00:00:00Z', processingMode: 'manual', targets: [] };
const PREVIEW = {
  requestId: 7, generatedAt: '2026-09-13T01:00:00Z', planDigest: DIGEST, blockers: [], files: ['uploads/profile/42.jpg'], unhandled: [],
  tables: [
    { table: 'WEO_BOARDBBS', action: 'anonymize', columns: ['SEQ', 'USR_SEQ', 'SUBJECT'], maskedColumns: [], changedColumns: ['USR_SEQ', 'SUBJECT'], count: 1,
      rows: [{ before: ['10', '42', '원래 제목'], after: ['10', '0', '탈퇴한 회원의 삭제된 게시글입니다.'] }] },
    { table: 'WEO_MEMBER', action: 'delete', columns: ['USR_SEQ', 'USR_PASS'], maskedColumns: ['USR_PASS'], changedColumns: [], count: 1, rows: [{ before: ['42', '[보안값 비표시]'] }] },
  ],
};

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><AccountDeletionsPage /></QueryClientProvider>);
}

function mockGet(previews: unknown[]) {
  let call = 0;
  return vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
    if (url.endsWith('/preview')) return previews[Math.min(call++, previews.length - 1)];
    return { items: [QUEUE_ITEM] };
  });
}

it('shows affected records with before and after values before any processing', async () => {
  mockGet([PREVIEW]);
  const put = vi.spyOn(api, 'put').mockResolvedValue(undefined);
  const user = userEvent.setup();
  mount();
  expect(screen.queryByRole('button', { name: '검토 완료 · 지금 탈퇴 처리' })).not.toBeInTheDocument();
  await user.click(await screen.findByRole('button', { name: '처리 대상 기록 불러오기' }));
  expect(await screen.findByText('작성한 게시글')).toBeInTheDocument();
  expect(screen.getByText(/익명화 · 행은 남기고 표시된 칸만 변경 · 1건/)).toBeInTheDocument();
  expect(screen.getByText('원래 제목')).toBeInTheDocument();
  expect(screen.getByText('탈퇴한 회원의 삭제된 게시글입니다.')).toBeInTheDocument();
  expect(screen.getByText(/행 삭제 · 1건/)).toBeInTheDocument();
  expect(screen.getByText('[보안값 비표시]')).toBeInTheDocument();
  expect(put).not.toHaveBeenCalled();
});

it('approves only after review confirmation, with the reviewed plan digest', async () => {
  mockGet([PREVIEW]);
  const put = vi.spyOn(api, 'put').mockResolvedValue(undefined);
  const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
  const user = userEvent.setup();
  mount();
  await user.click(await screen.findByRole('button', { name: '처리 대상 기록 불러오기' }));
  const approve = await screen.findByRole('button', { name: '검토 완료 · 지금 탈퇴 처리' });
  expect(approve).toBeDisabled();
  await user.click(screen.getByLabelText(/직접 확인했으며 영향 범위에 문제가 없습니다/));
  await user.click(approve);
  expect(put).not.toHaveBeenCalled();
  confirm.mockReturnValue(true);
  await user.click(approve);
  await waitFor(() => expect(put).toHaveBeenCalledWith('/api/admin/account-deletions/7', { action: 'expedite', reviewedPlanDigest: DIGEST }));
});

it('requires a fresh review when records changed after the preview', async () => {
  const changed = { ...PREVIEW, planDigest: 'b'.repeat(64) };
  const get = mockGet([PREVIEW, changed]);
  vi.spyOn(api, 'put').mockRejectedValue(new ApiClientError(409, 'ERASURE_PLAN_CHANGED', 'changed'));
  vi.spyOn(window, 'confirm').mockReturnValue(true);
  const user = userEvent.setup();
  mount();
  await user.click(await screen.findByRole('button', { name: '처리 대상 기록 불러오기' }));
  await user.click(await screen.findByLabelText(/직접 확인했으며 영향 범위에 문제가 없습니다/));
  await user.click(screen.getByRole('button', { name: '검토 완료 · 지금 탈퇴 처리' }));
  expect(await screen.findByText(/처리 대상 기록이 바뀌었습니다/)).toBeInTheDocument();
  await waitFor(() => expect(get.mock.calls.filter(([url]) => String(url).endsWith('/preview'))).toHaveLength(2));
  expect(screen.getByLabelText(/직접 확인했으며 영향 범위에 문제가 없습니다/)).not.toBeChecked();
  expect(screen.getByRole('button', { name: '검토 완료 · 지금 탈퇴 처리' })).toBeDisabled();
});
