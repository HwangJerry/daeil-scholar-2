import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { useActiveBannerAd } from '../hooks/useActiveBannerAd';
import type { BannerAd } from '../types/api';

vi.mock('../api/client', () => ({
  api: { get: vi.fn() },
}));

const activeBanner: BannerAd = {
  bnSeq: 7,
  bnName: '테스트 배너',
  bnUrl: 'https://example.com/banner',
  images: [
    {
      bniSeq: 71,
      bnSeq: 7,
      imageUrl: 'https://example.com/banner.jpg',
      sortOrder: 1,
    },
  ],
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('useActiveBannerAd', () => {
  it('returns the active banner', async () => {
    vi.mocked(api.get).mockResolvedValue(activeBanner);

    const { result } = renderHook(() => useActiveBannerAd(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toEqual(activeBanner);
    });
    expect(api.get).toHaveBeenCalledWith('/api/banner-ad/active');
  });

  it('returns null when there is no active banner', async () => {
    vi.mocked(api.get).mockResolvedValue(null);

    const { result } = renderHook(() => useActiveBannerAd(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
    expect(result.current.data).toBeNull();
    expect(api.get).toHaveBeenCalledWith('/api/banner-ad/active');
  });
});
