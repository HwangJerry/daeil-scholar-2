import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { FeedList } from '../components/feed/FeedList';

vi.mock('../hooks/useInfiniteScroll', () => ({
  useInfiniteScroll: () => ({ sentinelRef: vi.fn() }),
}));
vi.mock('../components/feed/AdCard', () => ({
  AdCard: () => <div>PUBLIC_AD_SHOULD_NOT_RENDER</div>,
}));
vi.mock('../hooks/useFeedPagination', () => ({
  useFeedPagination: () => ({
    items: [
      {
        type: 'notice',
        seq: 10,
        subject: '장학회 소식',
        summary: '후배들을 위한 장학사업 소식입니다.',
        thumbnailUrl: null,
        regDate: '2026-07-29T00:00:00Z',
        regName: '장학회',
        hit: 3,
        likeCnt: 2,
        commentCnt: 1,
        isPinned: 'N',
        userLiked: false,
      },
      {
        type: 'ad',
        maSeq: 20,
        maName: '광고주',
        maUrl: 'https://example.com',
        imageUrl: '',
        adTier: 'NORMAL',
        titleLabel: '광고',
        likeCnt: 0,
        commentCnt: 0,
        hit: 0,
        userLiked: false,
      },
    ],
    hasMore: false,
    isFetching: false,
    loadMore: vi.fn(),
  }),
}));

describe('public notice feed', () => {
  it('renders notices without ads, likes, or comments', () => {
    vi.stubGlobal('matchMedia', vi.fn().mockImplementation(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })));
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <FeedList />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText('장학회 소식')).toBeInTheDocument();
    expect(screen.queryByText('PUBLIC_AD_SHOULD_NOT_RENDER')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '댓글 펼치기' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /좋아요/ })).not.toBeInTheDocument();
  });
});
