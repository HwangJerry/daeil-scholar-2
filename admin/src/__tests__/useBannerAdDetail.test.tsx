import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiClientError } from '../api/client.ts';
import { useBannerAdDetail } from '../hooks/useBannerAdDetail.ts';
import type { AdminBannerAdRow } from '../types/api.ts';

const BANNER_AD_SEQ = 12;

const BANNER_AD: AdminBannerAdRow = {
  bnSeq: BANNER_AD_SEQ,
  bnName: '동문 행사 안내',
  bnUrl: 'https://example.com/event',
  openYn: 'N',
  indx: 2,
  bnStartDate: null,
  bnEndDate: null,
  createdAt: '2026-08-01T00:00:00Z',
  updatedAt: '2026-08-02T00:00:00Z',
  images: [],
  viewCount: 19,
  clickCount: 3,
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('useBannerAdDetail', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('returns a banner ad for an existing sequence', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => BANNER_AD,
    });
    vi.stubGlobal('fetch', fetchMock);

    const { result } = renderHook(() => useBannerAdDetail(BANNER_AD_SEQ), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(fetchMock).toHaveBeenCalledWith(
      `/api/admin/banner-ad/${BANNER_AD_SEQ}`,
      expect.objectContaining({ method: 'GET' }),
    );
    expect(result.current.data).toEqual(BANNER_AD);
  });

  it('exposes a 404 API error for a missing banner ad', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      statusText: 'Not Found',
      json: async () => ({ code: 'BANNER_AD_NOT_FOUND', message: 'banner ad not found' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    const { result } = renderHook(() => useBannerAdDetail(BANNER_AD_SEQ), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(ApiClientError);
    expect(result.current.error).toEqual(expect.objectContaining({
      status: 404,
      code: 'BANNER_AD_NOT_FOUND',
    }));
  });
});
