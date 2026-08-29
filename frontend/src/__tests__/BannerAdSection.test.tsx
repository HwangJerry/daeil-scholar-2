import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { BannerAdSection } from '../components/feed/BannerAdSection';
import type { BannerAd } from '../types/api';

vi.mock('../api/client', () => ({
  api: { get: vi.fn(), post: vi.fn() },
}));

vi.mock('../hooks/useBannerAdImpression', () => ({
  useBannerAdImpression: vi.fn(() => ({ ref: vi.fn() })),
}));

const singleImageBanner: BannerAd = {
  bnSeq: 7,
  bnName: '단일 이미지 배너',
  bnUrl: 'https://example.com/single',
  images: [
    {
      bniSeq: 71,
      bnSeq: 7,
      imageUrl: 'https://example.com/single.jpg',
      sortOrder: 1,
    },
  ],
};

const multipleImageBanner: BannerAd = {
  bnSeq: 8,
  bnName: '여러 이미지 배너',
  bnUrl: 'https://example.com/multiple',
  images: [
    {
      bniSeq: 81,
      bnSeq: 8,
      imageUrl: 'https://example.com/first.jpg',
      sortOrder: 1,
    },
    {
      bniSeq: 82,
      bnSeq: 8,
      imageUrl: 'https://example.com/second.jpg',
      sortOrder: 2,
    },
  ],
};

function renderBannerAdSection() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return {
    queryClient,
    ...render(
      <QueryClientProvider client={queryClient}>
        <BannerAdSection />
      </QueryClientProvider>,
    ),
  };
}

describe('BannerAdSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.post).mockResolvedValue(undefined);
  });

  it('renders no element or layout space when there is no active banner', async () => {
    vi.mocked(api.get).mockResolvedValue(null);

    const { container, queryClient } = renderBannerAdSection();

    await waitFor(() => {
      expect(queryClient.getQueryState(['banner-ad', 'active'])?.status).toBe('success');
    });
    expect(container).toBeEmptyDOMElement();
  });

  it('renders a single image without carousel controls', async () => {
    vi.mocked(api.get).mockResolvedValue(singleImageBanner);

    renderBannerAdSection();

    expect(await screen.findByRole('img', { name: '단일 이미지 배너' })).toHaveAttribute(
      'src',
      'https://example.com/single.jpg',
    );
    expect(screen.queryByRole('button', { name: '이전 배너 이미지' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '1번 배너로 이동' })).not.toBeInTheDocument();
  });

  it('renders a carousel and pagination for multiple images', async () => {
    vi.mocked(api.get).mockResolvedValue(multipleImageBanner);

    renderBannerAdSection();

    expect(await screen.findByRole('region', { name: '여러 이미지 배너' })).toHaveAttribute(
      'aria-roledescription',
      'carousel',
    );
    expect(screen.getAllByRole('img', { name: '여러 이미지 배너' })).toHaveLength(2);
    expect(screen.getByRole('button', { name: '이전 배너 이미지' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '다음 배너 이미지' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '1번 배너로 이동' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.getByRole('button', { name: '2번 배너로 이동' })).toBeInTheDocument();
  });

  it('records a click when the banner link is clicked', async () => {
    vi.mocked(api.get).mockResolvedValue(singleImageBanner);

    renderBannerAdSection();

    fireEvent.click(await screen.findByRole('link', { name: '단일 이미지 배너' }));

    expect(api.post).toHaveBeenCalledWith('/api/banner-ad/7/click');
  });
});
