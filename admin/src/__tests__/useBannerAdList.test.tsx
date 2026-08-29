import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useBannerAdList } from '../hooks/useBannerAdList.ts';
import { useToast } from '../hooks/useToast.ts';
import type { AdminBannerAdListItem } from '../types/api.ts';

const BANNER_AD_SEQ = 7;

const BANNER_ADS: AdminBannerAdListItem[] = [{
  bnSeq: BANNER_AD_SEQ,
  bnName: '장학재단 앱 설치',
  bnUrl: 'https://example.com/app',
  openYn: 'Y',
  indx: 1,
  bnStartDate: '2026-08-01T00:00:00Z',
  bnEndDate: '2026-08-31T23:59:59Z',
  createdAt: '2026-07-20T00:00:00Z',
  updatedAt: '2026-07-21T00:00:00Z',
  images: [{
    bniSeq: 11,
    bnSeq: BANNER_AD_SEQ,
    imageUrl: '/uploads/banner/app.png',
    sortOrder: 1,
  }],
  viewCount: 321,
  clickCount: 45,
}];

function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('useBannerAdList', () => {
  beforeEach(() => useToast.setState({ toasts: [] }));
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('returns the banner ad list including view and click counts', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => BANNER_ADS,
    });
    vi.stubGlobal('fetch', fetchMock);
    const queryClient = createQueryClient();

    const { result } = renderHook(() => useBannerAdList(), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/admin/banner-ad',
      expect.objectContaining({ method: 'GET' }),
    );
    expect(result.current.data).toEqual(BANNER_ADS);
    expect(result.current.data?.[0]).toEqual(expect.objectContaining({
      viewCount: 321,
      clickCount: 45,
    }));
  });

  it('invalidates the banner ad list after a successful deletion', async () => {
    const fetchMock = vi.fn(async (_url: string, options?: RequestInit) => {
      if (options?.method === 'DELETE') {
        return { ok: true, status: 204 };
      }

      return {
        ok: true,
        status: 200,
        json: async () => BANNER_ADS,
      };
    });
    vi.stubGlobal('fetch', fetchMock);
    const queryClient = createQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useBannerAdList(), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    act(() => result.current.deleteAd(BANNER_AD_SEQ));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        `/api/admin/banner-ad/${BANNER_AD_SEQ}`,
        expect.objectContaining({ method: 'DELETE' }),
      );
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ['admin', 'bannerAds'] });
    });
  });
});
