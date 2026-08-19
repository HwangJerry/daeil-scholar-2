import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DonationOrdersSection } from '../components/donation/DonationOrdersSection.tsx';
import type { DonationOrderPage } from '../types/api.ts';

const ORDER_PAGE: DonationOrderPage = {
  items: [{
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
  }],
  total: 1,
  page: 1,
  size: 20,
};

function renderSection() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <DonationOrdersSection />
    </QueryClientProvider>,
  );
}

describe('DonationOrdersSection', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('renders canonical order fields and sends canonical filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ORDER_PAGE,
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    renderSection();

    expect(await screen.findByText('홍길동')).toBeInTheDocument();
    expect(screen.getByText('30기 / 경영학과')).toBeInTheDocument();
    expect(screen.getByText('010-1234-5678')).toBeInTheDocument();
    expect(screen.getByText('#7')).toBeInTheDocument();

    await user.type(screen.getByRole('textbox', { name: '기부자명 검색' }), '김');

    expect(fetchMock).toHaveBeenLastCalledWith(
      expect.stringContaining('name=%EA%B9%80'),
      expect.objectContaining({ method: 'GET' }),
    );
  });
});
