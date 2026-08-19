import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DonationOrderForm } from '../components/donation/DonationOrderForm.tsx';
import type { DonationOrder } from '../types/api.ts';

const EDITING_ORDER: DonationOrder = {
  orderSeq: 42,
  accountUsrSeq: 7,
  source: 'bank_transfer',
  transactionNumber: 'TX-42',
  donationDate: '2026-08-20',
  donor: { name: '홍길동', cohort: '30기', department: '경영학과', phone: '01012345678' },
  donationType: 'one_time',
  grossAmount: 50000,
  refundedAmount: 0,
  netReceivedAmount: 50000,
  status: 'completed',
  paymentMethod: 'bank',
  memo: null,
  lastEditedBy: 1,
  lastEditedAt: '2026-08-20T12:00:00Z',
  lastEditedIp: '127.0.0.1',
};

describe('DonationOrderForm', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('shows canonical field errors before sending an invalid order', async () => {
    const queryClient = new QueryClient();
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    render(
      <QueryClientProvider client={queryClient}>
        <DonationOrderForm open onOpenChange={vi.fn()} />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole('button', { name: '등록' }));

    expect(screen.getByText('기부자명을 입력해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('기수를 입력해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('학과를 입력해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('010-1234-5678 또는 01012345678 형식으로 입력해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('총액은 0 이상의 정수여야 합니다.')).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('sends lastEditedAt and explains a concurrent administrator conflict', async () => {
    const queryClient = new QueryClient();
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      statusText: 'Conflict',
      json: async () => ({
        code: 'DONATION_ORDER_STALE',
        message: '다른 관리자가 먼저 수정했습니다.',
      }),
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    render(
      <QueryClientProvider client={queryClient}>
        <DonationOrderForm open order={EDITING_ORDER} onOpenChange={vi.fn()} />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole('button', { name: '수정 저장' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해주세요.',
    );
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/admin/donation/orders/42',
      expect.objectContaining({
        method: 'PUT',
        body: expect.stringContaining('"lastEditedAt":"2026-08-20T12:00:00Z"'),
      }),
    );
  });
});
