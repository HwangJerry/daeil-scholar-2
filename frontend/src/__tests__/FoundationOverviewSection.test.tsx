// FoundationOverviewSection.test — Landing foundation overview data-state contracts
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { FoundationOverviewSection } from '../components/landing/FoundationOverviewSection';
import { EXTERNAL_DONATION_URL } from '../constants/donation';

vi.mock('../api/client', () => ({
  api: { get: vi.fn() },
}));

function renderSection() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }

  return render(<FoundationOverviewSection />, { wrapper: Wrapper });
}

describe('FoundationOverviewSection', () => {
  it('renders the foundation message and canonical cumulative donation amount', async () => {
    vi.mocked(api.get).mockResolvedValue({ displayAmount: 123_456_789 });

    renderSection();

    const section = document.getElementById('about');
    expect(section).toHaveClass('scroll-mt-[var(--landing-header-height)]');
    expect(
      screen.getByRole('heading', { name: /동문의 마음으로,\s*후배의 내일을 엽니다/ }),
    ).toBeInTheDocument();
    expect(screen.getByText(/후배들의 꿈과 성장을 꾸준히 지원/)).toBeInTheDocument();
    expect(await screen.findByText('1억 2,345만원')).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith('/api/donation/summary');

    const donationLink = screen.getByRole('link', { name: '기부하기' });
    expect(donationLink).toHaveAttribute('href', EXTERNAL_DONATION_URL);
    expect(donationLink).toHaveAttribute('target', '_blank');
  });

  it('renders a skeleton while the donation summary is loading', () => {
    vi.mocked(api.get).mockReturnValue(new Promise(() => undefined));

    renderSection();

    expect(screen.getByRole('status', { name: '누적 기부액 불러오는 중' })).toBeInTheDocument();
  });

  it('renders an accessible message when the donation summary request fails', async () => {
    vi.mocked(api.get).mockRejectedValue(new Error('network unavailable'));

    renderSection();

    expect(await screen.findByRole('alert')).toHaveTextContent('기부 현황을 불러오지 못했습니다.');
  });
});
