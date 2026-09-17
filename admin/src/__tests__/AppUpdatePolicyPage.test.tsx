// AppUpdatePolicyPage.test — per-platform saves, rollout confirmation, and server error surfacing
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { ApiClientError, api } from '../api/client';
import { AppUpdatePolicyPage } from '../pages/AppUpdatePolicyPage';
import type { AppUpdatePolicy } from '../types/appUpdate';

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const IOS_UPDATED_AT = '2026-09-16T10:00:00Z';
const ANDROID_UPDATED_AT = '2026-09-16T09:00:00Z';
const IOS_BUILD = 202609151230;
const ANDROID_BUILD = 260915001;

const DISABLED_IOS_POLICY: AppUpdatePolicy = {
  forceEnabled: false,
  minBuild: 0,
  recommendEnabled: false,
  recommendedBuild: 0,
  minOsVersion: '17.0',
  storeUrl: 'https://apps.apple.com/app/id1234567890',
};

function stubReads(iosPolicy: AppUpdatePolicy = DISABLED_IOS_POLICY) {
  return vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
    if (url.startsWith('/api/admin/app-client-builds')) {
      const platform = url.endsWith('ios') ? 'ios' : 'android';
      return {
        builds: [
          {
            platform,
            build: platform === 'ios' ? IOS_BUILD : ANDROID_BUILD,
            versionName: '1.0.0',
            firstSeenAt: IOS_UPDATED_AT,
            lastSeenAt: IOS_UPDATED_AT,
          },
        ],
      } as never;
    }
    if (url.endsWith('/history')) {
      return { entries: [] } as never;
    }
    return {
      policies: [
        { platform: 'ios', policy: iosPolicy, updatedAt: IOS_UPDATED_AT, updatedBy: 7 },
        {
          platform: 'android',
          policy: { ...DISABLED_IOS_POLICY, minOsVersion: '26', storeUrl: undefined },
          updatedAt: ANDROID_UPDATED_AT,
          updatedBy: null,
        },
      ],
    } as never;
  });
}

function mount() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <AppUpdatePolicyPage />
    </QueryClientProvider>,
  );
}

async function iosCard() {
  return (await screen.findByRole('heading', { name: 'iOS' })).closest('article') as HTMLElement;
}

it('requires rollout confirmation before enabling a force update', async () => {
  stubReads();
  const put = vi.spyOn(api, 'put').mockResolvedValue(undefined as never);
  const user = userEvent.setup();
  mount();

  const card = await iosCard();
  await user.click(within(card).getByLabelText('강제 업데이트 사용'));
  await user.selectOptions(
    within(card).getByLabelText('최소 빌드 (이 빌드 미만이면 차단)'),
    String(IOS_BUILD),
  );
  await user.click(within(card).getByRole('button', { name: 'iOS 저장' }));

  const dialog = await screen.findByRole('alertdialog');
  expect(put).not.toHaveBeenCalled();
  expect(within(dialog).getByRole('button', { name: '적용' })).toBeDisabled();

  await user.click(within(dialog).getByRole('checkbox'));
  await user.click(within(dialog).getByRole('button', { name: '적용' }));

  await waitFor(() =>
    expect(put).toHaveBeenCalledWith('/api/admin/app-update-policies/ios', {
      policy: { ...DISABLED_IOS_POLICY, forceEnabled: true, minBuild: IOS_BUILD },
      updatedAt: IOS_UPDATED_AT,
      expectedPolicy: DISABLED_IOS_POLICY,
      allowUnobservedBuild: false,
    }),
  );
});

it('saves one platform without touching the other', async () => {
  stubReads();
  const put = vi.spyOn(api, 'put').mockResolvedValue(undefined as never);
  const user = userEvent.setup();
  mount();

  const card = await iosCard();
  await user.click(within(card).getByLabelText('권장 업데이트 사용'));
  await user.selectOptions(
    within(card).getByLabelText('권장 빌드 (이 빌드 미만이면 안내)'),
    String(IOS_BUILD),
  );
  await user.click(within(card).getByRole('button', { name: 'iOS 저장' }));

  await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
  expect(put.mock.calls[0][0]).toBe('/api/admin/app-update-policies/ios');
  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
});

it('sends the android policy with its own updatedAt', async () => {
  stubReads();
  const put = vi.spyOn(api, 'put').mockResolvedValue(undefined as never);
  const user = userEvent.setup();
  mount();

  const androidCard = (await screen.findByRole('heading', { name: 'Android' })).closest(
    'article',
  ) as HTMLElement;
  await user.clear(within(androidCard).getByLabelText('최소 지원 API 레벨'));
  await user.type(within(androidCard).getByLabelText('최소 지원 API 레벨'), '28');
  await user.click(within(androidCard).getByRole('button', { name: 'Android 저장' }));

  await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
  const [url, body] = put.mock.calls[0];
  expect(url).toBe('/api/admin/app-update-policies/android');
  expect(body).toMatchObject({ updatedAt: ANDROID_UPDATED_AT, policy: { minOsVersion: '28' } });
});

it('shows the server field message when a threshold is rejected', async () => {
  stubReads();
  vi.spyOn(api, 'put').mockRejectedValue(
    new ApiClientError(400, 'INVALID_POLICY', '정책 값을 저장할 수 없습니다', [], {
      code: 'INVALID_POLICY',
      message: '정책 값을 저장할 수 없습니다',
      details: { fields: [{ field: 'minBuild', reason: '이 플랫폼에서 확인된 적 없는 빌드입니다' }] },
    }),
  );
  const user = userEvent.setup();
  mount();

  const card = await iosCard();
  await user.click(within(card).getByLabelText('권장 업데이트 사용'));
  await user.click(within(card).getByRole('button', { name: 'iOS 저장' }));

  expect(await within(card).findByText('이 플랫폼에서 확인된 적 없는 빌드입니다')).toBeInTheDocument();
});

it('explains a concurrent edit instead of retrying it', async () => {
  stubReads();
  vi.spyOn(api, 'put').mockRejectedValue(
    new ApiClientError(409, 'POLICY_CONFLICT', '다른 관리자가 먼저 수정했습니다'),
  );
  const user = userEvent.setup();
  mount();

  const card = await iosCard();
  await user.click(within(card).getByLabelText('권장 업데이트 사용'));
  await user.click(within(card).getByRole('button', { name: 'iOS 저장' }));

  expect(
    await within(card).findByText('다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해 주세요.'),
  ).toBeInTheDocument();
});

it('keeps a readable platform when the other policy row is broken', async () => {
  vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
    if (url.startsWith('/api/admin/app-client-builds')) return { builds: [] } as never;
    if (url.endsWith('/history')) return { entries: [] } as never;
    return {
      policies: [
        { platform: 'ios', policy: DISABLED_IOS_POLICY, updatedAt: IOS_UPDATED_AT, updatedBy: 7 },
        {
          platform: 'android',
          policy: DISABLED_IOS_POLICY,
          updatedAt: ANDROID_UPDATED_AT,
          updatedBy: null,
          unavailable: true,
        },
      ],
    } as never;
  });
  mount();

  expect(await screen.findByRole('button', { name: 'iOS 저장' })).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Android 저장' })).not.toBeInTheDocument();
  expect(screen.getByText(/저장된 정책을 읽을 수 없습니다/)).toBeInTheDocument();
});

it('flags a manually typed build as unobserved, and clears the flag when an observed one is picked', async () => {
  stubReads();
  const put = vi.spyOn(api, 'put').mockResolvedValue(undefined as never);
  const user = userEvent.setup();
  mount();

  const card = await iosCard();
  await user.click(within(card).getByLabelText('권장 업데이트 사용'));
  const select = within(card).getByLabelText('권장 빌드 (이 빌드 미만이면 안내)');
  await user.selectOptions(select, 'manual');
  await user.type(within(card).getByLabelText('빌드 번호 직접 입력'), '999');
  await user.click(within(card).getByRole('button', { name: 'iOS 저장' }));

  await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
  expect(put.mock.calls[0][1]).toMatchObject({ allowUnobservedBuild: true });

  await user.selectOptions(select, String(IOS_BUILD));
  await user.click(within(card).getByRole('button', { name: 'iOS 저장' }));

  await waitFor(() => expect(put).toHaveBeenCalledTimes(2));
  expect(put.mock.calls[1][1]).toMatchObject({
    allowUnobservedBuild: false,
    policy: { recommendedBuild: IOS_BUILD },
  });
});

// Clearing the field must not unmount the input and snap the select back.
it('keeps the manual input mounted while the number is being retyped', async () => {
  stubReads();
  vi.spyOn(api, 'put').mockResolvedValue(undefined as never);
  const user = userEvent.setup();
  mount();

  const card = await iosCard();
  await user.selectOptions(within(card).getByLabelText('최소 빌드 (이 빌드 미만이면 차단)'), 'manual');
  const manualInput = within(card).getByLabelText('빌드 번호 직접 입력');
  await user.type(manualInput, '202609151230');
  await user.clear(manualInput);

  expect(within(card).getByLabelText('빌드 번호 직접 입력')).toBeInTheDocument();

  await user.type(within(card).getByLabelText('빌드 번호 직접 입력'), '202609151231');
  expect(within(card).getByLabelText('빌드 번호 직접 입력')).toHaveValue('202609151231');
});
