import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { DonationArchivesPage } from '../pages/DonationArchivesPage';
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
function mount() {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><DonationArchivesPage /></QueryClientProvider>);
}
it('requires a purpose and clears the requested plaintext on close', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({ items: [{ id: 7, basis: 'ledger_10y', retainUntil: '2030-12-31' }] });
  const read = vi.spyOn(api, 'post').mockResolvedValue({ donorName: '합성 기부자', donationDate: '2026-01-01', grossAmount: 10000, refundedAmount: 1000, netAmount: 9000, source: 'bank', transactionNumber: 'synthetic', basisDate: '2026-01-01', evidenceReference: 'test' });
  const user = userEvent.setup(); mount();
  const button = await screen.findByRole('button', { name: '증빙 열람' });
  expect(button).toBeDisabled(); expect(read).not.toHaveBeenCalled();
  await user.selectOptions(screen.getByRole('combobox'), 'accounting_review');
  await user.click(button);
  expect(await screen.findByText('합성 기부자')).toBeInTheDocument();
  expect(screen.getByText('10,000원 / 1,000원 / 9,000원')).toBeInTheDocument();
  expect(read).toHaveBeenCalledWith('/api/admin/donation-archives/7/read', { purpose: 'accounting_review' });
  await user.click(screen.getByRole('button', { name: '증빙 닫기' }));
  expect(screen.queryByText('합성 기부자')).not.toBeInTheDocument();
});
it('shows authorization failure without requesting plaintext', async () => {
  vi.spyOn(api, 'get').mockRejectedValue(new Error('최고 관리자 권한이 필요합니다.'));
  const read = vi.spyOn(api, 'post'); mount();
  expect(await screen.findByRole('alert')).toHaveTextContent('최고 관리자 권한');
  expect(read).not.toHaveBeenCalled();
});
