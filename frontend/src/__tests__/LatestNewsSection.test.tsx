// LatestNewsSection.test — Landing news data states and modal-navigation contracts
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { LatestNewsSection } from '../components/landing/LatestNewsSection';
import { useFeedPagination } from '../hooks/useFeedPagination';
import { useHeroNotice } from '../hooks/useHeroNotice';
import type { HeroNotice, NoticeItem } from '../types/api';

vi.mock('../hooks/useFeedPagination');
vi.mock('../hooks/useHeroNotice');
vi.mock('../hooks/useInfiniteScroll', () => ({
  useInfiniteScroll: () => ({ sentinelRef: vi.fn() }),
}));

const HERO_NOTICE: HeroNotice = {
  seq: 1,
  subject: '대표 장학회 소식',
  summary: '대표 공지 요약입니다.',
  thumbnailUrl: null,
  regDate: '2026-08-20T00:00:00Z',
  regName: '장학회',
  hit: 10,
  likeCnt: 0,
  commentCnt: 0,
  isPinned: 'Y',
};

const NOTICE_ITEMS: NoticeItem[] = Array.from({ length: 5 }, (_, index) => ({
  type: 'notice',
  seq: index + 2,
  subject: `일반 공지 ${index + 1}`,
  summary: `일반 공지 ${index + 1} 요약입니다.`,
  thumbnailUrl: index === 0 ? '/notice-thumbnail.jpg' : null,
  regDate: `2026-08-${String(19 - index).padStart(2, '0')}T00:00:00Z`,
  regName: '장학회',
  hit: index,
  likeCnt: 0,
  commentCnt: 0,
  isPinned: 'N',
  userLiked: false,
}));

const refetchHero = vi.fn();
const refetchFeed = vi.fn();
const loadMore = vi.fn();

function mockHeroQuery(overrides: Record<string, unknown> = {}) {
  vi.mocked(useHeroNotice).mockReturnValue({
    data: HERO_NOTICE,
    isError: false,
    isLoading: false,
    refetch: refetchHero,
    ...overrides,
  } as unknown as ReturnType<typeof useHeroNotice>);
}

function mockFeedQuery(overrides: Record<string, unknown> = {}) {
  vi.mocked(useFeedPagination).mockReturnValue({
    items: NOTICE_ITEMS,
    hasMore: false,
    isError: false,
    isFetching: false,
    loadMore,
    refetch: refetchFeed,
    ...overrides,
  } as ReturnType<typeof useFeedPagination>);
}

function LocationProbe() {
  const location = useLocation();
  const state = location.state as { backgroundLocation?: { pathname: string } } | null;

  return (
    <output data-testid="location-state">
      {location.pathname}|{state?.backgroundLocation?.pathname ?? 'none'}
    </output>
  );
}

function renderSection() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <LatestNewsSection />
      <LocationProbe />
    </MemoryRouter>,
  );
}

describe('LatestNewsSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('matchMedia', vi.fn().mockImplementation(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })));
    mockHeroQuery();
    mockFeedQuery();
  });

  it('renders a four-item editorial preview without duplicating the hero notice', () => {
    mockFeedQuery({ items: [
      { ...NOTICE_ITEMS[0], seq: HERO_NOTICE.seq, subject: HERO_NOTICE.subject },
      ...NOTICE_ITEMS,
    ] });

    renderSection();

    expect(document.getElementById('news')).toHaveClass(
      'scroll-mt-[var(--landing-header-height)]',
    );
    expect(screen.getByRole('heading', { name: '최근 소식' })).toBeInTheDocument();
    expect(screen.getAllByText(HERO_NOTICE.subject)).toHaveLength(1);
    expect(screen.getByText('일반 공지 4')).toBeInTheDocument();
    expect(screen.queryByText('일반 공지 5')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /좋아요|댓글/ })).not.toBeInTheDocument();
  });

  it('progressively reveals more already-loaded notices', async () => {
    const user = userEvent.setup();
    mockFeedQuery({ hasMore: true });
    renderSection();

    expect(screen.queryByText('일반 공지 5')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '소식 더 보기' }));

    expect(screen.getByText('일반 공지 5')).toBeInTheDocument();
    expect(loadMore).not.toHaveBeenCalled();
  });

  it('preserves desktop background-location navigation for notice details', async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(screen.getByRole('link', { name: /일반 공지 1/ }));

    expect(screen.getByTestId('location-state')).toHaveTextContent('/post/2|/');
  });

  it('renders hero and list skeleton states while data is loading', () => {
    mockHeroQuery({ data: undefined, isLoading: true });
    mockFeedQuery({ items: [], isFetching: true });
    renderSection();

    expect(screen.getByRole('status', { name: '대표 소식 불러오는 중' })).toBeInTheDocument();
    expect(
      screen.getByRole('status', { name: '최신 소식 목록 불러오는 중' }),
    ).toBeInTheDocument();
  });

  it('renders empty states for both the hero and notice list', () => {
    mockHeroQuery({ data: undefined });
    mockFeedQuery({ items: [] });
    renderSection();

    expect(screen.getByRole('heading', { name: '대표 공지가 없습니다' })).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: '새로운 소식을 준비하고 있습니다' }),
    ).toBeInTheDocument();
  });

  it('renders retryable API error states for both queries', () => {
    mockHeroQuery({ data: undefined, isError: true });
    mockFeedQuery({ items: [], isError: true });
    renderSection();

    expect(
      screen.getByRole('heading', { name: '대표 소식을 불러오지 못했습니다' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: '최신 소식을 불러오지 못했습니다' }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: '다시 시도' })).toHaveLength(2);
  });
});
