import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { useBannerAdImpression } from '../hooks/useBannerAdImpression';
import { useInView } from 'react-intersection-observer';

vi.mock('../api/client', () => ({
  api: { post: vi.fn() },
}));

vi.mock('react-intersection-observer', () => ({
  useInView: vi.fn(),
}));

let isInView = false;

describe('useBannerAdImpression', () => {
  beforeEach(() => {
    isInView = false;
    vi.clearAllMocks();
    vi.mocked(api.post).mockResolvedValue(undefined);
    vi.mocked(useInView).mockImplementation(() => {
      const ref = vi.fn();
      return Object.assign([ref, isInView, undefined] as [typeof ref, boolean, undefined], {
        ref,
        inView: isInView,
        entry: undefined,
      });
    });
  });

  it('records a view only once for the same banner', async () => {
    const { rerender } = renderHook(({ bnSeq }) => useBannerAdImpression(bnSeq), {
      initialProps: { bnSeq: 11 },
    });

    isInView = true;
    rerender({ bnSeq: 11 });

    await waitFor(() => {
      expect(api.post).toHaveBeenCalledTimes(1);
    });
    expect(api.post).toHaveBeenCalledWith('/api/banner-ad/11/view');

    isInView = false;
    rerender({ bnSeq: 11 });
    isInView = true;
    rerender({ bnSeq: 11 });

    expect(api.post).toHaveBeenCalledTimes(1);
  });

  it('records a new view when the banner changes', async () => {
    isInView = true;
    const { rerender } = renderHook(({ bnSeq }) => useBannerAdImpression(bnSeq), {
      initialProps: { bnSeq: 11 },
    });

    await waitFor(() => {
      expect(api.post).toHaveBeenCalledWith('/api/banner-ad/11/view');
    });

    rerender({ bnSeq: 22 });

    await waitFor(() => {
      expect(api.post).toHaveBeenCalledWith('/api/banner-ad/22/view');
    });
    expect(api.post).toHaveBeenCalledTimes(2);
  });
});
