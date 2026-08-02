import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { DonationBanner } from '../components/feed/DonationBanner';

vi.mock('../api/client', () => ({
  api: { get: vi.fn() },
}));

describe('DonationBanner', () => {
  it('renders only the canonical cumulative net amount', async () => {
    vi.mocked(api.get).mockResolvedValue({ displayAmount: 123_456_789 });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    const { container } = render(
      <QueryClientProvider client={queryClient}>
        <DonationBanner />
      </QueryClientProvider>,
    );

    expect(await screen.findByText('1억 2,345만원')).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith('/api/donation/summary');
    expect(container.querySelector('#donation-summary')).toBeInTheDocument();
    expect(screen.queryByText(/명 참여/)).not.toBeInTheDocument();
    expect(screen.queryByText(/달성률/)).not.toBeInTheDocument();
    expect(screen.queryByText(/목표/)).not.toBeInTheDocument();
  });
});
