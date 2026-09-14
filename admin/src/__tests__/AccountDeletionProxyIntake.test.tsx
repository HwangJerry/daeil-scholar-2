// AccountDeletionProxyIntake.test — Proxy intake needs identity evidence and confirmation, and shows tokens once.
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { AccountDeletionProxyIntake } from '../components/AccountDeletionProxyIntake';

afterEach(() => { cleanup(); vi.restoreAllMocks(); });

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><AccountDeletionProxyIntake /></QueryClientProvider>);
}

it('registers only after evidence and confirmation, then shows both numbers', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({ receipt: { requestId: 9 }, receiptToken: 'a'.repeat(64), cancelToken: 'b'.repeat(64) });
  const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
  const user = userEvent.setup();
  mount();
  const submit = screen.getByRole('button', { name: '본인 확인 후 대신 접수' });
  await user.type(screen.getByLabelText('회원 번호'), '42');
  expect(submit).toBeDisabled();
  await user.type(screen.getByLabelText(/본인 확인 근거/), '가입 이메일 요청 확인 09-14');
  await user.click(submit);
  expect(post).not.toHaveBeenCalled();
  confirm.mockReturnValue(true);
  await user.click(submit);
  await waitFor(() => expect(post).toHaveBeenCalledWith('/api/admin/account-deletions', { userSeq: 42, evidenceReference: '가입 이메일 요청 확인 09-14' }));
  expect(await screen.findByDisplayValue('a'.repeat(64))).toBeInTheDocument();
  expect(screen.getByDisplayValue('b'.repeat(64))).toBeInTheDocument();
});
