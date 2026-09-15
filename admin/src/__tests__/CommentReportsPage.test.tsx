import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { CommentReportsPage } from '../pages/CommentReportsPage';

afterEach(() => { cleanup(); vi.restoreAllMocks(); });
it('requires a reason and confirmation before hiding a comment and renders evidence as text', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({ items: [{ id: 7, postId: 1, commentId: 11, reporterSeq: 2, reportedSeq: 3, reason: 'spam', details: '', content: '<script>unsafe()</script>', status: 'open', moderatorNote: '', createdAt: '2026-09-15T00:00:00Z', visible: true }] });
  const put = vi.spyOn(api, 'put').mockResolvedValue(undefined);
  const user = userEvent.setup();
  const { container } = render(<MemoryRouter><QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><CommentReportsPage /></QueryClientProvider></MemoryRouter>);
  expect(await screen.findByText('<script>unsafe()</script>')).toBeInTheDocument();
  expect(container.querySelector('script')).toBeNull();
  const hide = screen.getByRole('button', { name: '위반 댓글 숨김' });
  expect(hide).toBeDisabled();
  await user.type(screen.getByLabelText('처리 사유 (필수)'), '반복 광고');
  await user.click(hide);
  expect(put).not.toHaveBeenCalled();
  expect(screen.getByRole('alertdialog')).toBeInTheDocument();
  await user.click(screen.getByRole('button', { name: '확인' }));
  await waitFor(() => expect(put).toHaveBeenCalledWith('/api/admin/comment-reports/7', { status: 'removed', note: '반복 광고' }));
});
