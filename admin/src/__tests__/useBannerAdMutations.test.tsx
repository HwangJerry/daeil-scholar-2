import { MutationCache, QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiClientError } from '../api/client.ts';
import { useBannerAdMutations } from '../hooks/useBannerAdMutations.ts';
import { useToast } from '../hooks/useToast.ts';
import type { AdminBannerAdSaveRequest } from '../types/api.ts';

const BANNER_AD_SEQ = 17;

const SAVE_REQUEST: AdminBannerAdSaveRequest = {
  bnName: '여름 캠페인',
  bnUrl: 'https://example.com/summer',
  openYn: 'Y',
  indx: 1,
  bnStartDate: '2026-08-01T00:00:00Z',
  bnEndDate: '2026-08-31T23:59:59Z',
  imageUrls: ['/uploads/banner/summer.png'],
};

function createQueryClient(mutationCache?: MutationCache) {
  return new QueryClient({
    mutationCache,
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function BannerAdMutationWrapper({ children }: PropsWithChildren) {
    return (
      <MemoryRouter initialEntries={['/banner-ad/new']}>
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      </MemoryRouter>
    );
  };
}

describe('useBannerAdMutations', () => {
  beforeEach(() => useToast.setState({ toasts: [] }));
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('creates a banner ad and invalidates the list on success', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 });
    vi.stubGlobal('fetch', fetchMock);
    const queryClient = createQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useBannerAdMutations(undefined), {
      wrapper: createWrapper(queryClient),
    });

    act(() => result.current.save(SAVE_REQUEST));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/admin/banner-ad',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(SAVE_REQUEST),
        }),
      );
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ['admin', 'bannerAds'] });
    });
  });

  it('updates the selected banner ad and invalidates the list on success', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 });
    vi.stubGlobal('fetch', fetchMock);
    const queryClient = createQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useBannerAdMutations(BANNER_AD_SEQ), {
      wrapper: createWrapper(queryClient),
    });

    act(() => result.current.save(SAVE_REQUEST));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        `/api/admin/banner-ad/${BANNER_AD_SEQ}`,
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(SAVE_REQUEST),
        }),
      );
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ['admin', 'bannerAds'] });
    });
  });

  it('forwards a 400 validation error to mutation observers', async () => {
    const onMutationError = vi.fn();
    const mutationCache = new MutationCache({ onError: onMutationError });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      statusText: 'Bad Request',
      json: async () => ({
        code: 'INVALID_BANNER_AD',
        message: 'invalid banner ad',
        errors: [{ field: 'bnUrl', message: 'invalid URL' }],
      }),
    });
    vi.stubGlobal('fetch', fetchMock);
    const queryClient = createQueryClient(mutationCache);
    const { result } = renderHook(() => useBannerAdMutations(undefined), {
      wrapper: createWrapper(queryClient),
    });

    act(() => result.current.save(SAVE_REQUEST));

    await waitFor(() => expect(onMutationError).toHaveBeenCalledOnce());

    const receivedError: unknown = onMutationError.mock.calls[0]?.[0];
    expect(receivedError).toBeInstanceOf(ApiClientError);
    expect(receivedError).toEqual(expect.objectContaining({
      status: 400,
      code: 'INVALID_BANNER_AD',
      details: [{ field: 'bnUrl', message: 'invalid URL' }],
    }));
  });
});
