// HistorySection.test — Landing history API and UI-state contracts
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { HelmetProvider } from 'react-helmet-async';
import type { ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { HistorySection } from '../components/landing/HistorySection';
import type { HistoryYearGroup } from '../hooks/useHistory';
import { HistoryPage } from '../pages/HistoryPage';

vi.mock('../api/client', () => ({
  api: { get: vi.fn() },
}));

const HISTORY_GROUPS: HistoryYearGroup[] = [
  {
    year: 2025,
    items: [
      {
        heSeq: 12,
        eventDate: '2025-11-03',
        text: '후배 지원을 위한 새로운 장학사업을 시작했습니다.',
        sortOrder: 1,
      },
    ],
  },
  {
    year: 2024,
    items: [
      {
        heSeq: 11,
        eventDate: '2024-05-18',
        text: '동문 장학 네트워크를 확대했습니다.',
        sortOrder: 1,
      },
    ],
  },
];

const LONG_HISTORY_GROUPS: HistoryYearGroup[] = Array.from({ length: 5 }, (_, index) => {
  const year = 2025 - index;

  return {
    year,
    items: [
      {
        heSeq: 20 - index,
        eventDate: `${year}-01-01`,
        text: `${year}년 장학회 연혁`,
        sortOrder: 1,
      },
    ],
  };
});

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  });
}

function QueryWrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={createQueryClient()}>{children}</QueryClientProvider>;
}

function renderSection() {
  return render(<HistorySection />, { wrapper: QueryWrapper });
}

function renderHistoryPage() {
  const queryClient = createQueryClient();

  return render(
    <HelmetProvider>
      <MemoryRouter initialEntries={['/history']}>
        <QueryClientProvider client={queryClient}>
          <HistoryPage />
        </QueryClientProvider>
      </MemoryRouter>
    </HelmetProvider>,
  );
}

describe('HistorySection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders GET /api/history response data as a year-grouped timeline', async () => {
    vi.mocked(api.get).mockResolvedValue(HISTORY_GROUPS);

    renderSection();

    expect(document.getElementById('history')).toHaveClass(
      'scroll-mt-[var(--landing-header-height)]',
    );
    expect(await screen.findByText('후배 지원을 위한 새로운 장학사업을 시작했습니다.'))
      .toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '2025' })).toBeInTheDocument();
    expect(screen.getByText('11.03')).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledTimes(1);
    expect(api.get).toHaveBeenCalledWith('/api/history');
  });

  it('reveals all remaining year groups without another API request', async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockResolvedValue(LONG_HISTORY_GROUPS);

    renderSection();

    expect(await screen.findByText('2023년 장학회 연혁')).toBeInTheDocument();
    expect(screen.queryByText('2022년 장학회 연혁')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '연혁 더 보기' }));

    expect(screen.getByText('2022년 장학회 연혁')).toBeInTheDocument();
    expect(screen.getByText('2021년 장학회 연혁')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '연혁 더 보기' })).not.toBeInTheDocument();
    expect(api.get).toHaveBeenCalledTimes(1);
  });

  it('does not render the expand button when all year groups are initially visible', async () => {
    vi.mocked(api.get).mockResolvedValue(HISTORY_GROUPS);

    renderSection();

    expect(await screen.findByText('동문 장학 네트워크를 확대했습니다.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '연혁 더 보기' })).not.toBeInTheDocument();
  });

  it('renders a timeline-shaped skeleton while the API request is pending', () => {
    vi.mocked(api.get).mockReturnValue(new Promise(() => undefined));

    renderSection();

    const loadingState = screen.getByRole('status', { name: '연혁 불러오는 중' });
    expect(loadingState).toBeInTheDocument();
    expect(loadingState.querySelectorAll('.skeleton-shimmer')).toHaveLength(21);
  });

  it('renders an empty state when the API returns no history', async () => {
    vi.mocked(api.get).mockResolvedValue([]);

    renderSection();

    expect(
      await screen.findByRole('heading', { name: '연혁을 준비하고 있습니다' }),
    ).toBeInTheDocument();
  });

  it('renders an error state and refetches when retry is selected', async () => {
    const user = userEvent.setup();
    vi.mocked(api.get)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce(HISTORY_GROUPS);

    renderSection();

    expect(
      await screen.findByRole('heading', { name: '연혁을 불러오지 못했습니다' }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '다시 시도' }));

    expect(await screen.findByText('동문 장학 네트워크를 확대했습니다.')).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledTimes(2);
  });

  it('keeps the existing history page rendering the shared API data', async () => {
    vi.mocked(api.get).mockResolvedValue(HISTORY_GROUPS);

    renderHistoryPage();

    expect(await screen.findByRole('heading', { name: '연혁', level: 1 })).toBeInTheDocument();
    expect(await screen.findByText('후배 지원을 위한 새로운 장학사업을 시작했습니다.'))
      .toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith('/api/history');
  });
});
