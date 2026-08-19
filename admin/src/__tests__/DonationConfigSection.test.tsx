import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { DonationConfigSection } from '../components/donation/DonationConfigSection.tsx';
import { useToast } from '../hooks/useToast.ts';
import type { DonationConfig } from '../types/api.ts';

const DONATION_CONFIG: DonationConfig = {
  dcSeq: 1,
  dcGoal: 250_000_000,
  dcManualAdj: 5_000,
  dcManualDonorCnt: 12,
  dcTierSproutMin: 10_000,
  dcTierSaplingMin: 50_000,
  dcTierTreeMin: 100_000,
  dcTierBloomingMin: 300_000,
  dcTierFruitingMin: 1_000_000,
  dcNote: '기부 설정',
  dcOverwrite: 'N',
  isActive: 'Y',
  regDate: '2026-08-20 12:00:00',
};

function renderSection() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <DonationConfigSection />
    </QueryClientProvider>,
  );
}

function donationConfigFetch(responseForPut?: { ok: boolean; status: number; json?: () => Promise<unknown> }) {
  return vi.fn(async (_url: string, options?: RequestInit) => {
    if (options?.method === 'PUT') {
      return responseForPut ?? { ok: true, status: 204 };
    }

    return {
      ok: true,
      status: 200,
      json: async () => DONATION_CONFIG,
    };
  });
}

describe('DonationConfigSection', () => {
  beforeEach(() => useToast.setState({ toasts: [] }));
  afterEach(() => vi.unstubAllGlobals());

  it('saves all donation tree tier thresholds', async () => {
    const fetchMock = donationConfigFetch();
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    renderSection();
    await user.click(await screen.findByRole('button', { name: '수정' }));

    expect(screen.getByLabelText('새싹 기준 금액 (원)')).toHaveValue(10_000);
    expect(screen.getByLabelText('어린 나무 기준 금액 (원)')).toHaveValue(50_000);
    expect(screen.getByLabelText('나무 기준 금액 (원)')).toHaveValue(100_000);
    expect(screen.getByLabelText('꽃이 핀 나무 기준 금액 (원)')).toHaveValue(300_000);
    expect(screen.getByLabelText('열매 맺은 나무 기준 금액 (원)')).toHaveValue(1_000_000);

    await user.click(screen.getByRole('button', { name: '저장' }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/admin/donation/config',
        expect.objectContaining({ method: 'PUT' }),
      );
    });

    const putCall = fetchMock.mock.calls.find(([, options]) => options?.method === 'PUT');
    const requestBody = JSON.parse(String(putCall?.[1]?.body));
    expect(requestBody).toEqual(expect.objectContaining({
      tierSproutMin: 10_000,
      tierSaplingMin: 50_000,
      tierTreeMin: 100_000,
      tierBloomingMin: 300_000,
      tierFruitingMin: 1_000_000,
    }));
    expect(useToast.getState().toasts).toEqual(expect.arrayContaining([
      expect.objectContaining({ variant: 'success', title: '기부 설정이 저장되었습니다.' }),
    ]));
  });

  it('blocks saving when tier thresholds are not strictly increasing', async () => {
    const fetchMock = donationConfigFetch();
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    renderSection();
    await user.click(await screen.findByRole('button', { name: '수정' }));
    const saplingInput = screen.getByLabelText('어린 나무 기준 금액 (원)');
    await user.clear(saplingInput);
    await user.type(saplingInput, '10000');
    await user.click(screen.getByRole('button', { name: '저장' }));

    expect(screen.getByRole('alert')).toHaveTextContent('각 단계 금액은 이전 단계보다 커야 합니다.');
    expect(fetchMock.mock.calls.some(([, options]) => options?.method === 'PUT')).toBe(false);
  });

  it('shows a clear message when the server rejects tier thresholds', async () => {
    const fetchMock = donationConfigFetch({
      ok: false,
      status: 400,
      json: async () => ({
        code: 'INVALID_TIER_THRESHOLDS',
        message: 'donation tier thresholds must be non-negative and strictly increasing',
      }),
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    renderSection();
    await user.click(await screen.findByRole('button', { name: '수정' }));
    await user.click(screen.getByRole('button', { name: '저장' }));

    await waitFor(() => {
      expect(useToast.getState().toasts).toEqual(expect.arrayContaining([
        expect.objectContaining({
          variant: 'error',
          title: '나무 성장 단계 저장 실패',
          description: '각 단계 금액은 0원 이상이며 이전 단계보다 커야 합니다.',
        }),
      ]));
    });
  });
});
